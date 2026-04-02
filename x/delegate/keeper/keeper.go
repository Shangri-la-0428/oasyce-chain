package keeper

import (
	"crypto/sha256"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/delegate/types"
)

// Keeper manages the delegate module's state.
type Keeper struct {
	cdc        codec.Codec
	storeKey   storetypes.StoreKey
	bankKeeper types.BankKeeper
	router     baseapp.MessageRouter
	authority  string
}

// NewKeeper creates a new delegate Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	router baseapp.MessageRouter,
	authority string,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
		router:     router,
		authority:  authority,
	}
}

func (k Keeper) Authority() string { return k.authority }

// ---------------------------------------------------------------------------
// Policy CRUD
// ---------------------------------------------------------------------------

func (k Keeper) GetPolicy(ctx sdk.Context, principal string) (types.DelegatePolicy, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PolicyKey(principal))
	if bz == nil {
		return types.DelegatePolicy{}, false
	}
	var policy types.DelegatePolicy
	if err := k.cdc.Unmarshal(bz, &policy); err != nil {
		return types.DelegatePolicy{}, false
	}
	return policy, true
}

func (k Keeper) SetPolicy(ctx sdk.Context, policy types.DelegatePolicy) error {
	bz, err := k.cdc.Marshal(&policy)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.PolicyKey(policy.Principal), bz)
	return nil
}

func (k Keeper) DeletePolicy(ctx sdk.Context, principal string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PolicyKey(principal))
}

// IsPolicyExpired checks if a policy has expired based on block time.
func (k Keeper) IsPolicyExpired(ctx sdk.Context, policy types.DelegatePolicy) bool {
	if policy.ExpirationSeconds == 0 {
		return false // no expiry
	}
	now := ctx.BlockTime().Unix()
	return now > policy.CreatedAtSeconds+int64(policy.ExpirationSeconds)
}

// ---------------------------------------------------------------------------
// Delegate CRUD
// ---------------------------------------------------------------------------

func (k Keeper) GetDelegate(ctx sdk.Context, delegate string) (types.DelegateRecord, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.DelegateKey(delegate))
	if bz == nil {
		return types.DelegateRecord{}, false
	}
	var rec types.DelegateRecord
	if err := k.cdc.Unmarshal(bz, &rec); err != nil {
		return types.DelegateRecord{}, false
	}
	return rec, true
}

func (k Keeper) SetDelegate(ctx sdk.Context, rec types.DelegateRecord) error {
	bz, err := k.cdc.Marshal(&rec)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.DelegateKey(rec.Delegate), bz)
	// Reverse index for listing by principal
	store.Set(types.PrincipalDelegateKey(rec.Principal, rec.Delegate), []byte{})
	return nil
}

func (k Keeper) DeleteDelegate(ctx sdk.Context, principal, delegate string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.DelegateKey(delegate))
	store.Delete(types.PrincipalDelegateKey(principal, delegate))
}

func (k Keeper) ListDelegates(ctx sdk.Context, principal string) []types.DelegateRecord {
	store := ctx.KVStore(k.storeKey)
	prefix := types.PrincipalDelegateIteratorKey(principal)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var records []types.DelegateRecord
	for ; iter.Valid(); iter.Next() {
		// Key: prefix + principal + "/" + delegate_addr
		key := iter.Key()
		delegateAddr := string(key[len(prefix):])
		rec, found := k.GetDelegate(ctx, delegateAddr)
		if found {
			records = append(records, rec)
		}
	}
	return records
}

func (k Keeper) IterateAllDelegates(ctx sdk.Context, cb func(rec types.DelegateRecord) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.DelegateKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var rec types.DelegateRecord
		if err := k.cdc.Unmarshal(iter.Value(), &rec); err != nil {
			continue
		}
		if cb(rec) {
			break
		}
	}
}

func (k Keeper) IterateAllPolicies(ctx sdk.Context, cb func(policy types.DelegatePolicy) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.PolicyKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var policy types.DelegatePolicy
		if err := k.cdc.Unmarshal(iter.Value(), &policy); err != nil {
			continue
		}
		if cb(policy) {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// Spend Window
// ---------------------------------------------------------------------------

func (k Keeper) GetSpendWindow(ctx sdk.Context, principal string) (types.SpendWindow, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SpendKey(principal))
	if bz == nil {
		return types.SpendWindow{}, false
	}
	var w types.SpendWindow
	if err := k.cdc.Unmarshal(bz, &w); err != nil {
		return types.SpendWindow{}, false
	}
	return w, true
}

func (k Keeper) SetSpendWindow(ctx sdk.Context, w types.SpendWindow) error {
	bz, err := k.cdc.Marshal(&w)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.SpendKey(w.Principal), bz)
	return nil
}

// GetOrResetWindow returns the current spend window, resetting if expired.
func (k Keeper) GetOrResetWindow(ctx sdk.Context, principal string, windowSeconds uint64) types.SpendWindow {
	w, found := k.GetSpendWindow(ctx, principal)
	now := ctx.BlockTime().Unix()

	if !found || now >= w.WindowStart+int64(windowSeconds) {
		// Window expired or first use — reset.
		return types.SpendWindow{
			Principal:   principal,
			WindowStart: now,
			Spent:       sdk.NewCoin("uoas", math.ZeroInt()),
		}
	}
	return w
}

// ---------------------------------------------------------------------------
// Enrollment verification
// ---------------------------------------------------------------------------

// VerifyToken checks that sha256(token) matches the stored hash.
func VerifyToken(token string, storedHash []byte) bool {
	h := sha256.Sum256([]byte(token))
	if len(storedHash) != 32 {
		return false
	}
	for i := 0; i < 32; i++ {
		if h[i] != storedHash[i] {
			return false
		}
	}
	return true
}

// HashToken returns sha256(token).
func HashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// ---------------------------------------------------------------------------
// Exec: the core delegation execution
// ---------------------------------------------------------------------------

// ExecDelegate executes inner messages on behalf of the principal.
// Tracks gross outflow (sum of per-message balance decreases) to prevent
// buy+sell masking from bypassing spend limits.
func (k Keeper) ExecDelegate(ctx sdk.Context, delegateAddr string, innerMsgs []sdk.Msg) ([][]byte, error) {
	// Look up delegate -> principal.
	rec, found := k.GetDelegate(ctx, delegateAddr)
	if !found {
		return nil, types.ErrDelegateNotFound.Wrapf("delegate %s is not enrolled", delegateAddr)
	}

	// Look up principal's policy.
	policy, found := k.GetPolicy(ctx, rec.Principal)
	if !found {
		return nil, types.ErrPolicyNotFound.Wrapf("no policy for principal %s", rec.Principal)
	}

	// Check policy not expired.
	if k.IsPolicyExpired(ctx, policy) {
		return nil, types.ErrPolicyExpired.Wrapf("policy for %s has expired", rec.Principal)
	}

	principalAddr, _ := sdk.AccAddressFromBech32(rec.Principal)

	// Build allowed message type set.
	allowedMsgs := make(map[string]bool, len(policy.AllowedMsgs))
	for _, m := range policy.AllowedMsgs {
		allowedMsgs[m] = true
	}

	// Validate inner messages: correct signer + allowed type.
	// Uses codec.GetMsgV1Signers (same approach as x/authz).
	for i, msg := range innerMsgs {
		msgTypeURL := sdk.MsgTypeURL(msg)
		if !allowedMsgs[msgTypeURL] {
			return nil, types.ErrMsgNotAllowed.Wrapf("msg[%d] type %s not in policy allowed_msgs", i, msgTypeURL)
		}

		signers, _, err := k.cdc.GetMsgV1Signers(msg)
		if err != nil {
			return nil, types.ErrSignerMismatch.Wrapf("msg[%d]: cannot extract signer: %v", i, err)
		}
		if len(signers) != 1 {
			return nil, types.ErrSignerMismatch.Wrapf("msg[%d]: expected 1 signer, got %d", i, len(signers))
		}
		if sdk.AccAddress(signers[0]).String() != rec.Principal {
			return nil, types.ErrSignerMismatch.Wrapf("msg[%d] signer %s must be principal %s", i, sdk.AccAddress(signers[0]), rec.Principal)
		}
	}

	// Execute in cache context (atomic rollback on failure).
	// Track gross outflow: sum of per-message balance decreases.
	// This prevents buy+sell in one Exec from masking actual spend.
	denom := policy.PerTxLimit.Denom
	cacheCtx, write := ctx.CacheContext()
	grossOutflow := math.ZeroInt()
	var results [][]byte

	for i, msg := range innerMsgs {
		preBal := k.bankKeeper.GetBalance(cacheCtx, principalAddr, denom)

		handler := k.router.Handler(msg)
		if handler == nil {
			return nil, fmt.Errorf("no handler for msg[%d] type %s", i, sdk.MsgTypeURL(msg))
		}
		resp, err := handler(cacheCtx, msg)
		if err != nil {
			return nil, fmt.Errorf("msg[%d] execution failed: %w", i, err)
		}
		results = append(results, resp.Data)

		postBal := k.bankKeeper.GetBalance(cacheCtx, principalAddr, denom)
		delta := preBal.Amount.Sub(postBal.Amount)
		if delta.IsPositive() {
			grossOutflow = grossOutflow.Add(delta)
		}
	}

	// Check per-tx limit against gross outflow.
	if grossOutflow.GT(policy.PerTxLimit.Amount) {
		return nil, types.ErrExceedsPerTxLimit.Wrapf(
			"tx gross outflow %s exceeds per_tx_limit %s",
			grossOutflow.String(), policy.PerTxLimit.Amount.String(),
		)
	}

	// Check window limit.
	window := k.GetOrResetWindow(cacheCtx, rec.Principal, policy.WindowSeconds)
	newTotal := window.Spent.Amount.Add(grossOutflow)
	if newTotal.GT(policy.WindowLimit.Amount) {
		return nil, types.ErrExceedsWindowLimit.Wrapf(
			"window spend would be %s, exceeds window_limit %s",
			newTotal.String(), policy.WindowLimit.Amount.String(),
		)
	}

	// All checks passed — commit cache and update spend window.
	window.Spent = sdk.NewCoin(denom, newTotal)
	if err := k.SetSpendWindow(cacheCtx, window); err != nil {
		return nil, err
	}
	write() // commit to parent context

	// Emit event.
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"delegate_exec",
		sdk.NewAttribute("delegate", delegateAddr),
		sdk.NewAttribute("principal", rec.Principal),
		sdk.NewAttribute("msg_count", fmt.Sprintf("%d", len(innerMsgs))),
		sdk.NewAttribute("gross_outflow", grossOutflow.String()),
		sdk.NewAttribute("window_total", newTotal.String()),
	))

	return results, nil
}
