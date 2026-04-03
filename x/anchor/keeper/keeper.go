package keeper

import (
	"encoding/hex"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/anchor/types"
)

// Keeper manages the anchor module's state.
type Keeper struct {
	cdc       codec.BinaryCodec
	storeKey  storetypes.StoreKey
	authority string
}

// NewKeeper creates a new anchor Keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	authority string,
) Keeper {
	return Keeper{
		cdc:       cdc,
		storeKey:  storeKey,
		authority: authority,
	}
}

// Authority returns the module authority address.
func (k Keeper) Authority() string {
	return k.authority
}

// ---------------------------------------------------------------------------
// Anchor CRUD
// ---------------------------------------------------------------------------

// GetAnchor retrieves an anchor record by trace_id.
func (k Keeper) GetAnchor(ctx sdk.Context, traceID []byte) (types.AnchorRecord, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AnchorKey(traceID))
	if bz == nil {
		return types.AnchorRecord{}, false
	}
	var record types.AnchorRecord
	if err := k.cdc.Unmarshal(bz, &record); err != nil {
		return types.AnchorRecord{}, false
	}
	return record, true
}

// IsAnchored checks whether a trace_id has been anchored.
func (k Keeper) IsAnchored(ctx sdk.Context, traceID []byte) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.AnchorKey(traceID))
}

// SetAnchor persists an anchor record and creates secondary indexes.
func (k Keeper) SetAnchor(ctx sdk.Context, record types.AnchorRecord) error {
	bz, err := k.cdc.Marshal(&record)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)

	// Primary key: trace_id -> AnchorRecord
	store.Set(types.AnchorKey(record.TraceId), bz)

	// Secondary index: capability -> trace_id
	store.Set(types.AnchorByCapKey(record.Capability, record.TraceId), record.TraceId)

	// Secondary index: node_pubkey -> trace_id
	store.Set(types.AnchorByNodeKey(record.NodePubkey, record.TraceId), record.TraceId)

	// Secondary index: sigil_id -> trace_id (optional, only if provided)
	if record.SigilId != "" {
		store.Set(types.AnchorBySigilKey(record.SigilId, record.TraceId), record.TraceId)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Business Logic
// ---------------------------------------------------------------------------

// AnchorTrace validates and stores a single anchor record.
// Returns true if anchored, false if skipped (duplicate).
func (k Keeper) AnchorTrace(ctx sdk.Context, msg *types.MsgAnchorTrace) (bool, error) {
	// Validate node_pubkey is 32 bytes (ed25519).
	if len(msg.NodePubkey) != 32 {
		return false, types.ErrInvalidSigner.Wrapf("node_pubkey must be 32 bytes, got %d", len(msg.NodePubkey))
	}
	// NOTE: Signer is the fee payer (secp256k1 Cosmos account).
	// Node identity is proven by trace_signature (ed25519) — verified off-chain by consumers.
	// We do NOT require signer == sha256(node_pubkey)[:20] because Cosmos addresses derive
	// from secp256k1 keys (ripemd160(sha256(pubkey))), making the check impossible to satisfy.

	// Check for duplicate anchor.
	if k.IsAnchored(ctx, msg.TraceId) {
		return false, nil // skip duplicate
	}

	// Create the anchor record.
	record := types.AnchorRecord{
		TraceId:        msg.TraceId,
		NodePubkey:     msg.NodePubkey,
		Capability:     msg.Capability,
		Outcome:        msg.Outcome,
		Timestamp:      msg.Timestamp,
		AnchorHeight:   ctx.BlockHeight(),
		TraceSignature: msg.TraceSignature,
		SigilId:        msg.SigilId,
	}

	if err := k.SetAnchor(ctx, record); err != nil {
		return false, err
	}

	event := sdk.NewEvent(
		"trace_anchored",
		sdk.NewAttribute("trace_id", hex.EncodeToString(msg.TraceId)),
		sdk.NewAttribute("node_pubkey", hex.EncodeToString(msg.NodePubkey)),
		sdk.NewAttribute("capability", msg.Capability),
		sdk.NewAttribute("outcome", fmt.Sprintf("%d", msg.Outcome)),
		sdk.NewAttribute("anchor_height", fmt.Sprintf("%d", ctx.BlockHeight())),
	)
	if msg.SigilId != "" {
		event = event.AppendAttributes(sdk.NewAttribute("sigil_id", msg.SigilId))
	}
	ctx.EventManager().EmitEvent(event)

	return true, nil
}

// ---------------------------------------------------------------------------
// Iterators
// ---------------------------------------------------------------------------

// GetAnchorsByCapability returns all anchor records for a given capability.
func (k Keeper) GetAnchorsByCapability(ctx sdk.Context, capability string, limit uint64) []types.AnchorRecord {
	store := ctx.KVStore(k.storeKey)
	prefix := types.AnchorByCapIteratorPrefix(capability)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var anchors []types.AnchorRecord
	var count uint64
	for ; iter.Valid(); iter.Next() {
		if limit > 0 && count >= limit {
			break
		}
		traceID := iter.Value()
		record, found := k.GetAnchor(ctx, traceID)
		if found {
			anchors = append(anchors, record)
			count++
		}
	}
	return anchors
}

// GetAnchorsByNode returns all anchor records for a given node pubkey.
func (k Keeper) GetAnchorsByNode(ctx sdk.Context, nodePubkey []byte, limit uint64) []types.AnchorRecord {
	store := ctx.KVStore(k.storeKey)
	prefix := types.AnchorByNodeIteratorPrefix(nodePubkey)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var anchors []types.AnchorRecord
	var count uint64
	for ; iter.Valid(); iter.Next() {
		if limit > 0 && count >= limit {
			break
		}
		traceID := iter.Value()
		record, found := k.GetAnchor(ctx, traceID)
		if found {
			anchors = append(anchors, record)
			count++
		}
	}
	return anchors
}

// GetAnchorsBySigil returns all anchor records for a given sigil ID.
func (k Keeper) GetAnchorsBySigil(ctx sdk.Context, sigilID string, limit uint64) []types.AnchorRecord {
	store := ctx.KVStore(k.storeKey)
	prefix := types.AnchorBySigilIteratorPrefix(sigilID)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var anchors []types.AnchorRecord
	var count uint64
	for ; iter.Valid(); iter.Next() {
		if limit > 0 && count >= limit {
			break
		}
		traceID := iter.Value()
		record, found := k.GetAnchor(ctx, traceID)
		if found {
			anchors = append(anchors, record)
			count++
		}
	}
	return anchors
}

// IterateAllAnchors iterates over all anchor records and calls the callback.
// Returning true from the callback stops iteration.
func (k Keeper) IterateAllAnchors(ctx sdk.Context, cb func(record types.AnchorRecord) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.AnchorKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var record types.AnchorRecord
		if err := k.cdc.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		if cb(record) {
			break
		}
	}
}
