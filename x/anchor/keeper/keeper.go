package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"

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

	return nil
}

// ---------------------------------------------------------------------------
// Business Logic
// ---------------------------------------------------------------------------

// AnchorTrace validates and stores a single anchor record.
// Returns true if anchored, false if skipped (duplicate).
func (k Keeper) AnchorTrace(ctx sdk.Context, msg *types.MsgAnchorTrace) (bool, error) {
	// Verify signer matches pubkey derivation: sha256(pubkey)[:20] -> bech32
	if err := k.verifySigner(msg.Signer, msg.NodePubkey); err != nil {
		return false, err
	}

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
	}

	if err := k.SetAnchor(ctx, record); err != nil {
		return false, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"trace_anchored",
		sdk.NewAttribute("trace_id", hex.EncodeToString(msg.TraceId)),
		sdk.NewAttribute("node_pubkey", hex.EncodeToString(msg.NodePubkey)),
		sdk.NewAttribute("capability", msg.Capability),
		sdk.NewAttribute("outcome", fmt.Sprintf("%d", msg.Outcome)),
		sdk.NewAttribute("anchor_height", fmt.Sprintf("%d", ctx.BlockHeight())),
	))

	return true, nil
}

// verifySigner checks that the signer address is derived from the pubkey.
// Derivation: sha256(pubkey)[:20] -> bech32("oasyce", ...)
func (k Keeper) verifySigner(signer string, pubkey []byte) error {
	hash := sha256.Sum256(pubkey)
	addrBytes := hash[:20]

	// Decode the signer's bech32 address to get the raw bytes.
	hrp, signerBytes, err := bech32.DecodeAndConvert(signer)
	if err != nil {
		return types.ErrInvalidSigner.Wrapf("cannot decode signer address: %s", err)
	}

	_ = hrp // we don't enforce HRP here since SDK already validates it

	// Compare the derived address bytes with the signer's address bytes.
	if len(signerBytes) != len(addrBytes) {
		return types.ErrInvalidSigner.Wrapf(
			"signer address length mismatch: expected %d, got %d",
			len(addrBytes), len(signerBytes),
		)
	}
	for i := range addrBytes {
		if addrBytes[i] != signerBytes[i] {
			return types.ErrInvalidSigner.Wrapf(
				"signer does not match sha256(pubkey)[:20]; expected %x, got %x",
				addrBytes, signerBytes,
			)
		}
	}

	return nil
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
