package keeper

import (
	"crypto/sha256"

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
func (k Keeper) GetOrResetWindow(ctx sdk.Context, principal string, windowSeconds uint64, denom string) types.SpendWindow {
	w, found := k.GetSpendWindow(ctx, principal)
	now := ctx.BlockTime().Unix()

	if !found || now >= w.WindowStart+int64(windowSeconds) {
		// Window expired or first use — reset.
		return types.SpendWindow{
			Principal:   principal,
			WindowStart: now,
			Spent:       sdk.NewCoin(denom, math.ZeroInt()),
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
