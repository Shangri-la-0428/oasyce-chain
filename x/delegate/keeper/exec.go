package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/delegate/types"
)

const defaultMaxMsgsPerExec = 16

// ExecContext carries validated state through the execution pipeline.
// Each stage reads from and enriches this context.
type ExecContext struct {
	Delegate      types.DelegateRecord
	Policy        types.DelegatePolicy
	PrincipalAddr sdk.AccAddress
	AllowedMsgs   map[string]bool
	InnerMsgs     []sdk.Msg
	Results       [][]byte
	GrossOutflow  math.Int
}

type legacySignerMsg interface {
	GetSigners() []sdk.AccAddress
}

func (k Keeper) extractSigners(msg sdk.Msg) ([]sdk.AccAddress, error) {
	signers, _, err := k.cdc.GetMsgV1Signers(msg)
	if err == nil {
		addrs := make([]sdk.AccAddress, len(signers))
		for i, signer := range signers {
			addrs[i] = sdk.AccAddress(signer)
		}
		return addrs, nil
	}

	if legacyMsg, ok := msg.(legacySignerMsg); ok {
		return legacyMsg.GetSigners(), nil
	}

	return nil, err
}

func (k Keeper) resolveAndAuthorize(ctx sdk.Context, delegateAddr string, ec *ExecContext) error {
	rec, found := k.GetDelegate(ctx, delegateAddr)
	if !found {
		return types.ErrDelegateNotFound.Wrapf("delegate %s is not enrolled", delegateAddr)
	}

	policy, found := k.GetPolicy(ctx, rec.Principal)
	if !found {
		return types.ErrPolicyNotFound.Wrapf("no policy for principal %s", rec.Principal)
	}
	if k.IsPolicyExpired(ctx, policy) {
		return types.ErrPolicyExpired.Wrapf("policy for %s has expired", rec.Principal)
	}

	principalAddr, err := sdk.AccAddressFromBech32(rec.Principal)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("invalid principal: %v", err)
	}

	allowedMsgs := make(map[string]bool, len(policy.AllowedMsgs))
	for _, msgType := range policy.AllowedMsgs {
		allowedMsgs[msgType] = true
	}

	ec.Delegate = rec
	ec.Policy = policy
	ec.PrincipalAddr = principalAddr
	ec.AllowedMsgs = allowedMsgs
	return nil
}

func (k Keeper) validateInnerMsgs(_ sdk.Context, ec *ExecContext) error {
	maxMsgsPerExec := defaultMaxMsgsPerExec
	if ec.Policy.MaxMsgsPerExec > 0 {
		maxMsgsPerExec = int(ec.Policy.MaxMsgsPerExec)
	}
	if len(ec.InnerMsgs) > maxMsgsPerExec {
		return types.ErrTooManyMessages.Wrapf(
			"got %d inner messages, max allowed is %d",
			len(ec.InnerMsgs), maxMsgsPerExec,
		)
	}

	for i, msg := range ec.InnerMsgs {
		msgTypeURL := sdk.MsgTypeURL(msg)
		if !ec.AllowedMsgs[msgTypeURL] {
			return types.ErrMsgNotAllowed.Wrapf("msg[%d] type %s not in policy allowed_msgs", i, msgTypeURL)
		}

		signers, err := k.extractSigners(msg)
		if err != nil {
			return types.ErrSignerMismatch.Wrapf("msg[%d]: cannot extract signer: %v", i, err)
		}
		if len(signers) != 1 {
			return types.ErrSignerMismatch.Wrapf("msg[%d]: expected 1 signer, got %d", i, len(signers))
		}
		if signers[0].String() != ec.Delegate.Principal {
			return types.ErrSignerMismatch.Wrapf(
				"msg[%d] signer %s must be principal %s",
				i,
				signers[0],
				ec.Delegate.Principal,
			)
		}
	}

	return nil
}

func (k Keeper) executeAndTrack(ctx sdk.Context, ec *ExecContext) error {
	denom := ec.Policy.PerTxLimit.Denom
	ec.GrossOutflow = math.ZeroInt()
	ec.Results = ec.Results[:0]

	for i, msg := range ec.InnerMsgs {
		preBal := k.bankKeeper.GetBalance(ctx, ec.PrincipalAddr, denom)

		handler := k.router.Handler(msg)
		if handler == nil {
			return fmt.Errorf("no handler for msg[%d] type %s", i, sdk.MsgTypeURL(msg))
		}

		resp, err := handler(ctx, msg)
		if err != nil {
			return fmt.Errorf("msg[%d] execution failed: %w", i, err)
		}
		ec.Results = append(ec.Results, resp.Data)

		postBal := k.bankKeeper.GetBalance(ctx, ec.PrincipalAddr, denom)
		delta := preBal.Amount.Sub(postBal.Amount)
		if delta.IsPositive() {
			ec.GrossOutflow = ec.GrossOutflow.Add(delta)
		}
	}

	return nil
}

func (k Keeper) enforceSpendLimits(ctx sdk.Context, ec *ExecContext, write func()) error {
	if ec.GrossOutflow.GT(ec.Policy.PerTxLimit.Amount) {
		return types.ErrExceedsPerTxLimit.Wrapf(
			"tx gross outflow %s exceeds per_tx_limit %s",
			ec.GrossOutflow.String(), ec.Policy.PerTxLimit.Amount.String(),
		)
	}

	window := k.GetOrResetWindow(ctx, ec.Delegate.Principal, ec.Policy.WindowSeconds, ec.Policy.PerTxLimit.Denom)
	newTotal := window.Spent.Amount.Add(ec.GrossOutflow)
	if newTotal.GT(ec.Policy.WindowLimit.Amount) {
		return types.ErrExceedsWindowLimit.Wrapf(
			"window spend would be %s, exceeds window_limit %s",
			newTotal.String(), ec.Policy.WindowLimit.Amount.String(),
		)
	}

	window.Spent = sdk.NewCoin(ec.Policy.PerTxLimit.Denom, newTotal)
	if err := k.SetSpendWindow(ctx, window); err != nil {
		return err
	}

	write()
	return nil
}

// ExecDelegate executes inner messages on behalf of the principal.
func (k Keeper) ExecDelegate(ctx sdk.Context, delegateAddr string, innerMsgs []sdk.Msg) ([][]byte, error) {
	ec := &ExecContext{InnerMsgs: innerMsgs}

	if err := k.resolveAndAuthorize(ctx, delegateAddr, ec); err != nil {
		return nil, err
	}
	if err := k.validateInnerMsgs(ctx, ec); err != nil {
		return nil, err
	}

	cacheCtx, write := ctx.CacheContext()
	if err := k.executeAndTrack(cacheCtx, ec); err != nil {
		return nil, err
	}
	if err := k.enforceSpendLimits(cacheCtx, ec, write); err != nil {
		return nil, err
	}

	window, _ := k.GetSpendWindow(ctx, ec.Delegate.Principal)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"delegate_exec",
		sdk.NewAttribute("delegate", delegateAddr),
		sdk.NewAttribute("principal", ec.Delegate.Principal),
		sdk.NewAttribute("msg_count", fmt.Sprintf("%d", len(innerMsgs))),
		sdk.NewAttribute("gross_outflow", ec.GrossOutflow.String()),
		sdk.NewAttribute("window_total", window.Spent.Amount.String()),
	))

	return ec.Results, nil
}
