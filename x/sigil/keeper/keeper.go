package keeper

import (
	"encoding/binary"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/sigil/types"
)

type Keeper struct {
	cdc       codec.BinaryCodec
	storeKey  storetypes.StoreKey
	authority string
}

func NewKeeper(cdc codec.BinaryCodec, storeKey storetypes.StoreKey, authority string) Keeper {
	return Keeper{cdc: cdc, storeKey: storeKey, authority: authority}
}

func (k Keeper) Authority() string { return k.authority }

func (k Keeper) StoreKey() storetypes.StoreKey { return k.storeKey }

func (k Keeper) TouchPulse(ctx sdk.Context, sigilID, dim string) error {
	s, found := k.GetSigil(ctx, sigilID)
	if !found {
		return types.ErrSigilNotFound.Wrapf("sigil %s", sigilID)
	}
	switch types.SigilStatus(s.Status) {
	case types.SigilStatusActive:
	case types.SigilStatusDissolved:
		return types.ErrSigilDissolved.Wrapf("sigil %s is dissolved", sigilID)
	case types.SigilStatusDormant:
		return types.ErrSigilNotActive.Wrapf("sigil %s is dormant, pulse rejected", sigilID)
	default:
		return types.ErrSigilNotActive.Wrapf("sigil %s is not active", sigilID)
	}
	if s.DimensionPulses == nil {
		s.DimensionPulses = make(map[string]int64)
	}
	height := ctx.BlockHeight()
	s.DimensionPulses[dim] = height
	s.LastActiveHeight = height
	return k.SetSigil(ctx, s)
}

// ---------------------------------------------------------------------------
// Sigil CRUD
// ---------------------------------------------------------------------------

func (k Keeper) GetSigil(ctx sdk.Context, sigilID string) (types.Sigil, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SigilKey(sigilID))
	if bz == nil {
		return types.Sigil{}, false
	}
	var s types.Sigil
	if err := k.cdc.Unmarshal(bz, &s); err != nil {
		return types.Sigil{}, false
	}
	return s, true
}

func (k Keeper) SetSigil(ctx sdk.Context, s types.Sigil) error {
	bz, err := k.cdc.Marshal(&s)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	var old *types.Sigil
	if existing, found := k.GetSigil(ctx, s.SigilId); found {
		old = &existing
	}

	// Primary key.
	store.Set(types.SigilKey(s.SigilId), bz)

	if old != nil {
		k.clearSigilIndexes(ctx, *old)
	}
	k.writeSigilIndexes(ctx, s)

	return nil
}

func (k Keeper) DeleteSigilFromLivenessIndex(ctx sdk.Context, height int64, sigilID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.LivenessIndexKey(height, sigilID))
}

func (k Keeper) DeleteSigilFromStatusIndex(ctx sdk.Context, status types.SigilStatus, sigilID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.SigilByStatusKey(status, sigilID))
}

// ---------------------------------------------------------------------------
// Bond CRUD
// ---------------------------------------------------------------------------

func (k Keeper) GetBond(ctx sdk.Context, bondID string) (types.Bond, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.BondKey(bondID))
	if bz == nil {
		return types.Bond{}, false
	}
	var b types.Bond
	if err := k.cdc.Unmarshal(bz, &b); err != nil {
		return types.Bond{}, false
	}
	return b, true
}

func (k Keeper) SetBond(ctx sdk.Context, b types.Bond) error {
	bz, err := k.cdc.Marshal(&b)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)

	// Primary key.
	store.Set(types.BondKey(b.BondId), bz)

	// BondsBySigil index (both directions).
	store.Set(types.BondsBySigilKey(b.SigilA, b.BondId), []byte(b.BondId))
	store.Set(types.BondsBySigilKey(b.SigilB, b.BondId), []byte(b.BondId))

	return nil
}

func (k Keeper) DeleteBond(ctx sdk.Context, b types.Bond) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.BondKey(b.BondId))
	store.Delete(types.BondsBySigilKey(b.SigilA, b.BondId))
	store.Delete(types.BondsBySigilKey(b.SigilB, b.BondId))
}

// ---------------------------------------------------------------------------
// Lineage
// ---------------------------------------------------------------------------

func (k Keeper) SetLineage(ctx sdk.Context, parentID, childID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.LineageKey(parentID, childID), []byte(childID))
}

func (k Keeper) GetChildren(ctx sdk.Context, parentID string) []string {
	store := ctx.KVStore(k.storeKey)
	prefix := types.LineageIteratorPrefix(parentID)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var children []string
	for ; iter.Valid(); iter.Next() {
		children = append(children, string(iter.Value()))
	}
	return children
}

// ---------------------------------------------------------------------------
// Bonds by Sigil
// ---------------------------------------------------------------------------

func (k Keeper) GetBondsBySigil(ctx sdk.Context, sigilID string) []types.Bond {
	store := ctx.KVStore(k.storeKey)
	prefix := types.BondsBySigilIteratorPrefix(sigilID)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var bonds []types.Bond
	for ; iter.Valid(); iter.Next() {
		bondID := string(iter.Value())
		b, found := k.GetBond(ctx, bondID)
		if found {
			bonds = append(bonds, b)
		}
	}
	return bonds
}

// ---------------------------------------------------------------------------
// Active count
// ---------------------------------------------------------------------------

func (k Keeper) GetActiveCount(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ActiveCountKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetActiveCount(ctx sdk.Context, count uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.ActiveCountKey, bz)
}

func (k Keeper) IncrementActiveCount(ctx sdk.Context) {
	k.SetActiveCount(ctx, k.GetActiveCount(ctx)+1)
}

func (k Keeper) DecrementActiveCount(ctx sdk.Context) {
	c := k.GetActiveCount(ctx)
	if c > 0 {
		k.SetActiveCount(ctx, c-1)
	}
}

// ---------------------------------------------------------------------------
// Params
// ---------------------------------------------------------------------------

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var p types.Params
	if err := k.cdc.Unmarshal(bz, &p); err != nil {
		return types.DefaultParams()
	}
	return p
}

func (k Keeper) SetParams(ctx sdk.Context, p types.Params) error {
	bz, err := k.cdc.Marshal(&p)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ParamsKey, bz)
	return nil
}

// ---------------------------------------------------------------------------
// Cross-module API
// ---------------------------------------------------------------------------

// RegisterSigil creates a new active Sigil from an external module (e.g. x/onboarding).
// Emits the same sigil_genesis event as MsgGenesis for a consistent event stream.
// Idempotent — returns existing ID if the pubkey already has a Sigil.
func (k Keeper) RegisterSigil(ctx sdk.Context, creator string, pubkey []byte, metadata string) (string, error) {
	sigilID := types.DeriveSigilID(pubkey)
	if _, found := k.GetSigil(ctx, sigilID); found {
		return sigilID, nil
	}
	sigil := types.Sigil{
		SigilId:          sigilID,
		Creator:          creator,
		PublicKey:        pubkey,
		Status:           types.SigilStatusActive,
		CreationHeight:   ctx.BlockHeight(),
		LastActiveHeight: ctx.BlockHeight(),
		Metadata:         metadata,
	}
	if err := k.SetSigil(ctx, sigil); err != nil {
		return "", err
	}
	k.IncrementActiveCount(ctx)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"sigil_genesis",
		sdk.NewAttribute("sigil_id", sigilID),
		sdk.NewAttribute("creator", creator),
		sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
	))

	return sigilID, nil
}

// ---------------------------------------------------------------------------
// Iterators
// ---------------------------------------------------------------------------

func (k Keeper) IterateAllSigils(ctx sdk.Context, cb func(s types.Sigil) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.SigilKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var s types.Sigil
		if err := k.cdc.Unmarshal(iter.Value(), &s); err != nil {
			continue
		}
		if cb(s) {
			break
		}
	}
}

func (k Keeper) IterateAllBonds(ctx sdk.Context, cb func(b types.Bond) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.BondKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var b types.Bond
		if err := k.cdc.Unmarshal(iter.Value(), &b); err != nil {
			continue
		}
		if cb(b) {
			break
		}
	}
}

// IterateStaleSigils iterates active sigils whose effective activity height
// (MaxPulseHeight) <= maxHeight via the liveness index.
func (k Keeper) IterateStaleSigils(ctx sdk.Context, maxHeight int64, cb func(sigilID string) bool) {
	store := ctx.KVStore(k.storeKey)
	endKey := types.LivenessIndexIteratorPrefix(maxHeight)
	iter := store.Iterator(types.LivenessIndexPrefix, endKey)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		sigilID := string(iter.Value())
		if cb(sigilID) {
			break
		}
	}
}
