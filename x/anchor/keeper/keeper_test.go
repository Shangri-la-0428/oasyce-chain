package keeper_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"

	"github.com/oasyce/chain/x/anchor/keeper"
	"github.com/oasyce/chain/x/anchor/types"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupKeeper creates a test keeper with an in-memory store.
func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	cms := store.NewCommitMultiStore(db, logger, metrics.NoOpMetrics{})
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}

	ctx := sdk.NewContext(cms, cmtproto.Header{Time: time.Now()}, false, logger)

	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)

	k := keeper.NewKeeper(cdc, storeKey, "authority")

	return k, ctx
}

// makePubkeyAndSigner creates a 32-byte test pubkey and its derived bech32
// signer address. The derivation matches the keeper's verifySigner logic:
// sha256(pubkey)[:20] -> bech32("oasyce", ...).
func makePubkeyAndSigner(seed string) ([]byte, string) {
	// Build a deterministic 32-byte pubkey from the seed.
	h := sha256.Sum256([]byte(seed))
	pubkey := h[:]

	// Derive the address the same way the keeper does.
	addrHash := sha256.Sum256(pubkey)
	addrBytes := addrHash[:20]

	signer, err := bech32.ConvertAndEncode("oasyce", addrBytes)
	if err != nil {
		panic(fmt.Sprintf("failed to encode bech32 address: %v", err))
	}

	return pubkey, signer
}

// makeSig returns a fake 64-byte signature (all zeros + seed byte).
func makeSig(b byte) []byte {
	sig := make([]byte, 64)
	sig[0] = b
	return sig
}

// makeTraceID returns a deterministic trace ID from a string seed.
func makeTraceID(seed string) []byte {
	h := sha256.Sum256([]byte(seed))
	return h[:32]
}

// validMsg builds a valid MsgAnchorTrace using the given pubkey/signer pair.
func validMsg(traceID []byte, pubkey []byte, signer string, capability string) *types.MsgAnchorTrace {
	return &types.MsgAnchorTrace{
		Signer:         signer,
		TraceId:        traceID,
		NodePubkey:     pubkey,
		Capability:     capability,
		Outcome:        1,
		Timestamp:      uint64(time.Now().Unix()),
		TraceSignature: makeSig(0x01),
	}
}

// ---------------------------------------------------------------------------
// 1. TestAnchorTrace — single trace anchoring success
// ---------------------------------------------------------------------------

func TestAnchorTrace(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	pubkey, signer := makePubkeyAndSigner("node-1")
	traceID := makeTraceID("trace-1")
	msg := validMsg(traceID, pubkey, signer, "text-generation")

	resp, err := ms.AnchorTrace(ctx, msg)
	if err != nil {
		t.Fatalf("AnchorTrace failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	// Verify the anchor is stored.
	record, found := k.GetAnchor(ctx, traceID)
	if !found {
		t.Fatal("anchor not found after AnchorTrace")
	}
	if !bytes.Equal(record.TraceId, traceID) {
		t.Fatalf("trace_id mismatch: expected %x, got %x", traceID, record.TraceId)
	}
	if !bytes.Equal(record.NodePubkey, pubkey) {
		t.Fatalf("node_pubkey mismatch")
	}
	if record.Capability != "text-generation" {
		t.Fatalf("capability mismatch: expected text-generation, got %s", record.Capability)
	}
	if record.Outcome != 1 {
		t.Fatalf("outcome mismatch: expected 1, got %d", record.Outcome)
	}
}

// ---------------------------------------------------------------------------
// 2. TestAnchorTrace_Duplicate — reject duplicate trace ID
// ---------------------------------------------------------------------------

func TestAnchorTrace_Duplicate(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	pubkey, signer := makePubkeyAndSigner("node-1")
	traceID := makeTraceID("trace-dup")

	msg := validMsg(traceID, pubkey, signer, "text-generation")

	// First anchor should succeed.
	_, err := ms.AnchorTrace(ctx, msg)
	if err != nil {
		t.Fatalf("first AnchorTrace failed: %v", err)
	}

	// Second anchor with same trace ID should fail.
	_, err = ms.AnchorTrace(ctx, msg)
	if err == nil {
		t.Fatal("expected duplicate anchor error, got nil")
	}
}

// ---------------------------------------------------------------------------
// 3. TestAnchorTrace_InvalidSigner — signer doesn't match pubkey derivation
// ---------------------------------------------------------------------------

func TestAnchorTrace_InvalidSigner(t *testing.T) {
	_, ctx := setupKeeper(t)
	k, _ := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	pubkey, _ := makePubkeyAndSigner("node-1")
	_, wrongSigner := makePubkeyAndSigner("node-other")

	traceID := makeTraceID("trace-invalid-signer")
	msg := validMsg(traceID, pubkey, wrongSigner, "text-generation")

	_, err := ms.AnchorTrace(ctx, msg)
	if err == nil {
		t.Fatal("expected invalid signer error, got nil")
	}
}

// ---------------------------------------------------------------------------
// 4. TestAnchorBatch — batch of 3 traces, all succeed
// ---------------------------------------------------------------------------

func TestAnchorBatch(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	pubkey, signer := makePubkeyAndSigner("batch-node")

	var anchors []*types.MsgAnchorTrace
	for i := 0; i < 3; i++ {
		traceID := makeTraceID(fmt.Sprintf("batch-trace-%d", i))
		anchors = append(anchors, validMsg(traceID, pubkey, signer, "image-recognition"))
	}

	batchMsg := &types.MsgAnchorBatch{
		Signer:  signer,
		Anchors: anchors,
	}

	resp, err := ms.AnchorBatch(ctx, batchMsg)
	if err != nil {
		t.Fatalf("AnchorBatch failed: %v", err)
	}
	if resp.Anchored != 3 {
		t.Fatalf("expected 3 anchored, got %d", resp.Anchored)
	}
	if resp.Skipped != 0 {
		t.Fatalf("expected 0 skipped, got %d", resp.Skipped)
	}

	// Verify each anchor is stored.
	for i := 0; i < 3; i++ {
		traceID := makeTraceID(fmt.Sprintf("batch-trace-%d", i))
		if !k.IsAnchored(ctx, traceID) {
			t.Fatalf("batch trace %d not found after AnchorBatch", i)
		}
	}
}

// ---------------------------------------------------------------------------
// 5. TestAnchorBatch_PartialDuplicate — batch where 1 is duplicate
// ---------------------------------------------------------------------------

func TestAnchorBatch_PartialDuplicate(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)

	pubkey, signer := makePubkeyAndSigner("partial-dup-node")

	// Pre-anchor one trace.
	preTraceID := makeTraceID("pre-existing-trace")
	preMsg := validMsg(preTraceID, pubkey, signer, "translation")
	_, err := ms.AnchorTrace(ctx, preMsg)
	if err != nil {
		t.Fatalf("pre-anchor failed: %v", err)
	}

	// Build a batch of 3 anchors: 1 duplicate + 2 new.
	var anchors []*types.MsgAnchorTrace
	// Duplicate.
	anchors = append(anchors, validMsg(preTraceID, pubkey, signer, "translation"))
	// New.
	anchors = append(anchors, validMsg(makeTraceID("new-trace-a"), pubkey, signer, "translation"))
	anchors = append(anchors, validMsg(makeTraceID("new-trace-b"), pubkey, signer, "translation"))

	batchMsg := &types.MsgAnchorBatch{
		Signer:  signer,
		Anchors: anchors,
	}

	resp, err := ms.AnchorBatch(ctx, batchMsg)
	if err != nil {
		t.Fatalf("AnchorBatch failed: %v", err)
	}
	if resp.Anchored != 2 {
		t.Fatalf("expected 2 anchored, got %d", resp.Anchored)
	}
	if resp.Skipped != 1 {
		t.Fatalf("expected 1 skipped, got %d", resp.Skipped)
	}
}

// ---------------------------------------------------------------------------
// 6. TestAnchorBatch_TooLarge — batch over 50, should fail validation
// ---------------------------------------------------------------------------

func TestAnchorBatch_TooLarge(t *testing.T) {
	pubkey, signer := makePubkeyAndSigner("large-batch-node")

	var anchors []*types.MsgAnchorTrace
	for i := 0; i < 51; i++ {
		traceID := makeTraceID(fmt.Sprintf("large-batch-trace-%d", i))
		anchors = append(anchors, validMsg(traceID, pubkey, signer, "text-generation"))
	}

	batchMsg := &types.MsgAnchorBatch{
		Signer:  signer,
		Anchors: anchors,
	}

	err := batchMsg.ValidateBasic()
	if err == nil {
		t.Fatal("expected batch too large validation error, got nil")
	}
}

// ---------------------------------------------------------------------------
// 7. TestQueryAnchor — get anchor by trace ID
// ---------------------------------------------------------------------------

func TestQueryAnchor(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)
	qs := keeper.NewQueryServer(k)

	pubkey, signer := makePubkeyAndSigner("query-node")
	traceID := makeTraceID("query-trace")
	msg := validMsg(traceID, pubkey, signer, "summarization")

	_, err := ms.AnchorTrace(ctx, msg)
	if err != nil {
		t.Fatalf("AnchorTrace failed: %v", err)
	}

	resp, err := qs.Anchor(ctx, &types.QueryAnchorRequest{TraceId: traceID})
	if err != nil {
		t.Fatalf("Query Anchor failed: %v", err)
	}
	if !bytes.Equal(resp.Anchor.TraceId, traceID) {
		t.Fatalf("trace_id mismatch in query response")
	}
	if resp.Anchor.Capability != "summarization" {
		t.Fatalf("capability mismatch: expected summarization, got %s", resp.Anchor.Capability)
	}
}

// ---------------------------------------------------------------------------
// 8. TestQueryAnchor_NotFound — query non-existent trace
// ---------------------------------------------------------------------------

func TestQueryAnchor_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServer(k)

	nonExistentID := makeTraceID("does-not-exist")
	_, err := qs.Anchor(ctx, &types.QueryAnchorRequest{TraceId: nonExistentID})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

// ---------------------------------------------------------------------------
// 9. TestQueryIsAnchored — check bool response
// ---------------------------------------------------------------------------

func TestQueryIsAnchored(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)
	qs := keeper.NewQueryServer(k)

	pubkey, signer := makePubkeyAndSigner("anchored-check-node")
	traceID := makeTraceID("anchored-check-trace")
	msg := validMsg(traceID, pubkey, signer, "code-review")

	// Before anchoring.
	resp, err := qs.IsAnchored(ctx, &types.QueryIsAnchoredRequest{TraceId: traceID})
	if err != nil {
		t.Fatalf("IsAnchored query failed: %v", err)
	}
	if resp.Anchored {
		t.Fatal("expected Anchored=false before anchoring")
	}

	// Anchor the trace.
	_, err = ms.AnchorTrace(ctx, msg)
	if err != nil {
		t.Fatalf("AnchorTrace failed: %v", err)
	}

	// After anchoring.
	resp, err = qs.IsAnchored(ctx, &types.QueryIsAnchoredRequest{TraceId: traceID})
	if err != nil {
		t.Fatalf("IsAnchored query failed: %v", err)
	}
	if !resp.Anchored {
		t.Fatal("expected Anchored=true after anchoring")
	}
}

// ---------------------------------------------------------------------------
// 10. TestQueryAnchorsByCapability — filter by capability with pagination
// ---------------------------------------------------------------------------

func TestQueryAnchorsByCapability(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)
	qs := keeper.NewQueryServer(k)

	pubkey, signer := makePubkeyAndSigner("cap-filter-node")

	// Anchor 5 traces with "text-generation" capability.
	for i := 0; i < 5; i++ {
		traceID := makeTraceID(fmt.Sprintf("cap-text-%d", i))
		msg := validMsg(traceID, pubkey, signer, "text-generation")
		if _, err := ms.AnchorTrace(ctx, msg); err != nil {
			t.Fatalf("AnchorTrace #%d failed: %v", i, err)
		}
	}

	// Anchor 3 traces with "image-recognition" capability.
	for i := 0; i < 3; i++ {
		traceID := makeTraceID(fmt.Sprintf("cap-image-%d", i))
		msg := validMsg(traceID, pubkey, signer, "image-recognition")
		if _, err := ms.AnchorTrace(ctx, msg); err != nil {
			t.Fatalf("AnchorTrace image #%d failed: %v", i, err)
		}
	}

	// Query text-generation (should return 5).
	resp, err := qs.AnchorsByCapability(ctx, &types.QueryAnchorsByCapabilityRequest{
		Capability: "text-generation",
	})
	if err != nil {
		t.Fatalf("AnchorsByCapability failed: %v", err)
	}
	if len(resp.Anchors) != 5 {
		t.Fatalf("expected 5 text-generation anchors, got %d", len(resp.Anchors))
	}
	for _, a := range resp.Anchors {
		if a.Capability != "text-generation" {
			t.Fatalf("expected capability text-generation, got %s", a.Capability)
		}
	}

	// Query image-recognition (should return 3).
	resp, err = qs.AnchorsByCapability(ctx, &types.QueryAnchorsByCapabilityRequest{
		Capability: "image-recognition",
	})
	if err != nil {
		t.Fatalf("AnchorsByCapability failed: %v", err)
	}
	if len(resp.Anchors) != 3 {
		t.Fatalf("expected 3 image-recognition anchors, got %d", len(resp.Anchors))
	}

	// Query with pagination limit of 2.
	resp, err = qs.AnchorsByCapability(ctx, &types.QueryAnchorsByCapabilityRequest{
		Capability: "text-generation",
		Pagination: &sdkquery.PageRequest{Limit: 2},
	})
	if err != nil {
		t.Fatalf("AnchorsByCapability with pagination failed: %v", err)
	}
	if len(resp.Anchors) != 2 {
		t.Fatalf("expected 2 anchors with limit=2, got %d", len(resp.Anchors))
	}

	// Query non-existent capability (should return 0).
	resp, err = qs.AnchorsByCapability(ctx, &types.QueryAnchorsByCapabilityRequest{
		Capability: "does-not-exist",
	})
	if err != nil {
		t.Fatalf("AnchorsByCapability for non-existent failed: %v", err)
	}
	if len(resp.Anchors) != 0 {
		t.Fatalf("expected 0 anchors for non-existent capability, got %d", len(resp.Anchors))
	}
}

// ---------------------------------------------------------------------------
// 11. TestQueryAnchorsByNode — filter by node pubkey with pagination
// ---------------------------------------------------------------------------

func TestQueryAnchorsByNode(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServer(k)
	qs := keeper.NewQueryServer(k)

	pubkey1, signer1 := makePubkeyAndSigner("node-alpha")
	pubkey2, signer2 := makePubkeyAndSigner("node-beta")

	// Anchor 4 traces from node-alpha.
	for i := 0; i < 4; i++ {
		traceID := makeTraceID(fmt.Sprintf("node-alpha-trace-%d", i))
		msg := validMsg(traceID, pubkey1, signer1, "text-generation")
		if _, err := ms.AnchorTrace(ctx, msg); err != nil {
			t.Fatalf("AnchorTrace alpha #%d failed: %v", i, err)
		}
	}

	// Anchor 2 traces from node-beta.
	for i := 0; i < 2; i++ {
		traceID := makeTraceID(fmt.Sprintf("node-beta-trace-%d", i))
		msg := validMsg(traceID, pubkey2, signer2, "text-generation")
		if _, err := ms.AnchorTrace(ctx, msg); err != nil {
			t.Fatalf("AnchorTrace beta #%d failed: %v", i, err)
		}
	}

	// Query node-alpha (should return 4).
	resp, err := qs.AnchorsByNode(ctx, &types.QueryAnchorsByNodeRequest{
		NodePubkey: pubkey1,
	})
	if err != nil {
		t.Fatalf("AnchorsByNode alpha failed: %v", err)
	}
	if len(resp.Anchors) != 4 {
		t.Fatalf("expected 4 anchors for node-alpha, got %d", len(resp.Anchors))
	}
	for _, a := range resp.Anchors {
		if !bytes.Equal(a.NodePubkey, pubkey1) {
			t.Fatalf("node_pubkey mismatch in result")
		}
	}

	// Query node-beta (should return 2).
	resp, err = qs.AnchorsByNode(ctx, &types.QueryAnchorsByNodeRequest{
		NodePubkey: pubkey2,
	})
	if err != nil {
		t.Fatalf("AnchorsByNode beta failed: %v", err)
	}
	if len(resp.Anchors) != 2 {
		t.Fatalf("expected 2 anchors for node-beta, got %d", len(resp.Anchors))
	}

	// Query with pagination limit of 2.
	resp, err = qs.AnchorsByNode(ctx, &types.QueryAnchorsByNodeRequest{
		NodePubkey: pubkey1,
		Pagination: &sdkquery.PageRequest{Limit: 2},
	})
	if err != nil {
		t.Fatalf("AnchorsByNode with pagination failed: %v", err)
	}
	if len(resp.Anchors) != 2 {
		t.Fatalf("expected 2 anchors with limit=2, got %d", len(resp.Anchors))
	}

	// Query non-existent node.
	unknownPubkey := make([]byte, 32)
	unknownPubkey[0] = 0xFF
	resp, err = qs.AnchorsByNode(ctx, &types.QueryAnchorsByNodeRequest{
		NodePubkey: unknownPubkey,
	})
	if err != nil {
		t.Fatalf("AnchorsByNode for unknown node failed: %v", err)
	}
	if len(resp.Anchors) != 0 {
		t.Fatalf("expected 0 anchors for unknown node, got %d", len(resp.Anchors))
	}
}
