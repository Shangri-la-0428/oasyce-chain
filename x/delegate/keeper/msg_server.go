package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/delegate/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// SetPolicy creates or updates a delegation policy.
// One command from the principal — all delegates operate under this.
func (m msgServer) SetPolicy(goCtx context.Context, msg *types.MsgSetPolicy) (*types.MsgSetPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	windowSeconds := msg.WindowSeconds
	if windowSeconds == 0 {
		windowSeconds = 86400 // default: 1 day
	}

	policy := types.DelegatePolicy{
		Principal:          msg.Principal,
		PerTxLimit:         msg.PerTxLimit,
		WindowLimit:        msg.WindowLimit,
		WindowSeconds:      windowSeconds,
		AllowedMsgs:        msg.AllowedMsgs,
		EnrollmentMode:     types.ENROLLMENT_MODE_TOKEN,
		EnrollmentTokenHash: HashToken(msg.EnrollmentToken),
		ExpirationSeconds:  msg.ExpirationSeconds,
		CreatedAtSeconds:   ctx.BlockTime().Unix(),
	}

	if err := m.Keeper.SetPolicy(ctx, policy); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"delegate_policy_set",
		sdk.NewAttribute("principal", msg.Principal),
		sdk.NewAttribute("allowed_msgs_count", fmt.Sprintf("%d", len(msg.AllowedMsgs))),
	))

	return &types.MsgSetPolicyResponse{}, nil
}

// Enroll self-registers an agent as a delegate.
// Agent signs with its own key. Zero human interaction.
func (m msgServer) Enroll(goCtx context.Context, msg *types.MsgEnroll) (*types.MsgEnrollResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check not already enrolled.
	if _, found := m.Keeper.GetDelegate(ctx, msg.Delegate); found {
		return nil, types.ErrAlreadyEnrolled.Wrapf("delegate %s already enrolled", msg.Delegate)
	}

	// Look up principal's policy.
	policy, found := m.Keeper.GetPolicy(ctx, msg.Principal)
	if !found {
		return nil, types.ErrPolicyNotFound.Wrapf("no policy for principal %s", msg.Principal)
	}

	// Check policy not expired.
	if m.Keeper.IsPolicyExpired(ctx, policy) {
		return nil, types.ErrPolicyExpired.Wrapf("policy for %s has expired", msg.Principal)
	}

	// Verify enrollment token.
	if policy.EnrollmentMode == types.ENROLLMENT_MODE_TOKEN {
		if !VerifyToken(msg.Token, policy.EnrollmentTokenHash) {
			return nil, types.ErrInvalidToken.Wrap("enrollment token does not match")
		}
	}

	// Create delegate record.
	rec := types.DelegateRecord{
		Delegate:          msg.Delegate,
		Principal:         msg.Principal,
		Label:             msg.Label,
		EnrolledAtSeconds: ctx.BlockTime().Unix(),
	}

	if err := m.Keeper.SetDelegate(ctx, rec); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"delegate_enrolled",
		sdk.NewAttribute("delegate", msg.Delegate),
		sdk.NewAttribute("principal", msg.Principal),
		sdk.NewAttribute("label", msg.Label),
	))

	return &types.MsgEnrollResponse{}, nil
}

// Revoke removes a delegate. Principal only.
func (m msgServer) Revoke(goCtx context.Context, msg *types.MsgRevoke) (*types.MsgRevokeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	rec, found := m.Keeper.GetDelegate(ctx, msg.Delegate)
	if !found {
		return nil, types.ErrDelegateNotFound.Wrapf("delegate %s not found", msg.Delegate)
	}
	if rec.Principal != msg.Principal {
		return nil, types.ErrDelegateNotFound.Wrapf("delegate %s not under principal %s", msg.Delegate, msg.Principal)
	}

	m.Keeper.DeleteDelegate(ctx, msg.Principal, msg.Delegate)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"delegate_revoked",
		sdk.NewAttribute("delegate", msg.Delegate),
		sdk.NewAttribute("principal", msg.Principal),
	))

	return &types.MsgRevokeResponse{}, nil
}

// Exec executes inner messages on behalf of the principal.
// Delegate signs. Chain validates policy + spend limits. Principal's balance moves.
func (m msgServer) Exec(goCtx context.Context, msg *types.MsgExec) (*types.MsgExecResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Unpack Any messages.
	innerMsgs := make([]sdk.Msg, len(msg.Msgs))
	for i, anyMsg := range msg.Msgs {
		var sdkMsg sdk.Msg
		if err := m.Keeper.cdc.UnpackAny(anyMsg, &sdkMsg); err != nil {
			return nil, fmt.Errorf("failed to unpack msg[%d]: %w", i, err)
		}
		innerMsgs[i] = sdkMsg
	}

	results, err := m.Keeper.ExecDelegate(ctx, msg.Delegate, innerMsgs)
	if err != nil {
		return nil, err
	}

	return &types.MsgExecResponse{Results: results}, nil
}
