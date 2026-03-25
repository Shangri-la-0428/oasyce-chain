package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/capability/types"
)

// Keeper manages the capability module state.
type Keeper struct {
	storeKey         storetypes.StoreKey
	cdc              codec.BinaryCodec
	bankKeeper       types.BankKeeper
	settlementKeeper types.SettlementKeeper
	authority        string
}

// NewKeeper creates a new capability Keeper.
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	bankKeeper types.BankKeeper,
	settlementKeeper types.SettlementKeeper,
	authority string,
) Keeper {
	return Keeper{
		storeKey:         storeKey,
		cdc:              cdc,
		bankKeeper:       bankKeeper,
		settlementKeeper: settlementKeeper,
		authority:        authority,
	}
}

// Authority returns the module authority address.
func (k Keeper) Authority() string {
	return k.authority
}

// --- Params ---

// GetParams returns the current module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams stores the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)
	return nil
}

// --- Counter helpers ---

func (k Keeper) getAndIncrementCounter(ctx sdk.Context, key []byte) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(key)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	store.Set(key, buf)
	return counter
}

func (k Keeper) nextCapabilityID(ctx sdk.Context) string {
	n := k.getAndIncrementCounter(ctx, types.CapabilityCounterKey)
	return fmt.Sprintf("CAP_%016x", n)
}

func (k Keeper) nextInvocationID(ctx sdk.Context) string {
	n := k.getAndIncrementCounter(ctx, types.InvocationCounterKey)
	return fmt.Sprintf("INV_%016x", n)
}

// --- Capability CRUD ---

func (k Keeper) setCapability(ctx sdk.Context, cap types.Capability) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&cap)
	if err != nil {
		return err
	}
	store.Set(types.CapabilityKey(cap.Id), bz)
	// Secondary index: provider -> capability
	store.Set(types.CapByProviderCapKey(cap.Provider, cap.Id), []byte(cap.Id))
	return nil
}

// GetCapability returns a capability by ID.
func (k Keeper) GetCapability(ctx sdk.Context, id string) (types.Capability, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CapabilityKey(id))
	if bz == nil {
		return types.Capability{}, types.ErrCapabilityNotFound.Wrapf("id: %s", id)
	}
	var cap types.Capability
	if err := k.cdc.Unmarshal(bz, &cap); err != nil {
		return types.Capability{}, err
	}
	return cap, nil
}

// ListCapabilities returns all capabilities, optionally filtered by tag.
func (k Keeper) ListCapabilities(ctx sdk.Context, tag string) []types.Capability {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.CapabilityKeyPrefix)
	defer iter.Close()

	var caps []types.Capability
	for ; iter.Valid(); iter.Next() {
		var cap types.Capability
		if err := k.cdc.Unmarshal(iter.Value(), &cap); err != nil {
			continue
		}
		if tag != "" {
			if !containsTag(cap.Tags, tag) {
				continue
			}
		}
		caps = append(caps, cap)
	}
	return caps
}

// ListByProvider returns all capabilities for a specific provider.
func (k Keeper) ListByProvider(ctx sdk.Context, provider string) []types.Capability {
	store := ctx.KVStore(k.storeKey)
	prefix := types.CapByProviderKey(provider)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var caps []types.Capability
	for ; iter.Valid(); iter.Next() {
		capID := string(iter.Value())
		cap, err := k.GetCapability(ctx, capID)
		if err != nil {
			continue
		}
		caps = append(caps, cap)
	}
	return caps
}

// RegisterCapability creates and stores a new capability.
func (k Keeper) RegisterCapability(ctx sdk.Context, msg *types.MsgRegisterCapability) (string, error) {
	params := k.GetParams(ctx)

	// Validate rate limit against max.
	if params.MaxRateLimit > 0 && msg.RateLimit > params.MaxRateLimit {
		return "", types.ErrRateLimitExceeded.Wrapf("requested %d exceeds max %d", msg.RateLimit, params.MaxRateLimit)
	}

	// Check minimum provider stake if configured.
	if params.MinProviderStake.IsPositive() {
		providerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
		if err != nil {
			return "", types.ErrInvalidInput.Wrapf("invalid provider address: %s", err)
		}
		spendable := k.bankKeeper.SpendableCoins(ctx, providerAddr)
		if spendable.AmountOf(params.MinProviderStake.Denom).LT(params.MinProviderStake.Amount) {
			return "", types.ErrInsufficientStake.Wrapf(
				"require %s, have %s",
				params.MinProviderStake,
				spendable.AmountOf(params.MinProviderStake.Denom),
			)
		}
	}

	id := k.nextCapabilityID(ctx)
	cap := types.Capability{
		Id:           id,
		Provider:     msg.Creator,
		Name:         msg.Name,
		Description:  msg.Description,
		EndpointUrl:  msg.EndpointUrl,
		PricePerCall: msg.PricePerCall,
		Tags:         msg.Tags,
		RateLimit:    msg.RateLimit,
		TotalCalls:   0,
		TotalEarned:  math.ZeroInt(),
		AvgLatencyMs: 0,
		SuccessRate:  10000, // 100% initially
		IsActive:     true,
		CreatedAt:    ctx.BlockTime(),
	}
	if err := k.setCapability(ctx, cap); err != nil {
		return "", err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"capability_registered",
		sdk.NewAttribute("capability_id", id),
		sdk.NewAttribute("provider", msg.Creator),
		sdk.NewAttribute("name", msg.Name),
	))

	return id, nil
}

// InvokeCapability creates an escrow and records a pending invocation.
func (k Keeper) InvokeCapability(ctx sdk.Context, msg *types.MsgInvokeCapability) (invocationID, escrowID string, err error) {
	cap, err := k.GetCapability(ctx, msg.CapabilityId)
	if err != nil {
		return "", "", err
	}

	if !cap.IsActive {
		return "", "", types.ErrInactive.Wrapf("capability %s is inactive", cap.Id)
	}

	// Compute input hash for the invocation record.
	inputHashBytes := sha256.Sum256(msg.Input)
	inputHash := hex.EncodeToString(inputHashBytes[:])

	// Create escrow for payment (skip for free capabilities).
	if cap.PricePerCall.IsPositive() {
		escrowID, err = k.settlementKeeper.CreateEscrow(
			ctx,
			msg.Creator,
			cap.Provider,
			cap.PricePerCall,
			300, // 5 minute timeout
		)
		if err != nil {
			return "", "", err
		}
	}

	invocationID = k.nextInvocationID(ctx)
	inv := types.Invocation{
		Id:           invocationID,
		CapabilityId: cap.Id,
		Consumer:     msg.Creator,
		Provider:     cap.Provider,
		InputHash:    inputHash,
		OutputHash:   "",
		Status:       types.StatusPending,
		Amount:       cap.PricePerCall,
		EscrowId:     escrowID,
		Timestamp:    ctx.BlockTime(),
	}
	if err := k.setInvocation(ctx, inv); err != nil {
		return "", "", err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"capability_invoked",
		sdk.NewAttribute("invocation_id", invocationID),
		sdk.NewAttribute("capability_id", cap.Id),
		sdk.NewAttribute("consumer", msg.Creator),
		sdk.NewAttribute("escrow_id", escrowID),
	))

	return invocationID, escrowID, nil
}

// CompleteInvocation marks an invocation as COMPLETED (awaiting challenge window).
// The escrow is NOT released yet — the provider must call ClaimInvocation after
// the challenge window passes, or the consumer can DisputeInvocation within it.
func (k Keeper) CompleteInvocation(ctx sdk.Context, invocationID, outputHash, caller, usageReport string) error {
	inv, err := k.GetInvocation(ctx, invocationID)
	if err != nil {
		return err
	}
	if inv.Status != types.StatusPending {
		return types.ErrInvalidStatus.Wrapf("invocation %s is not pending (status: %s)", invocationID, inv.Status)
	}
	// Only the provider can complete an invocation.
	if inv.Provider != caller {
		return types.ErrUnauthorized.Wrap("only the provider can complete an invocation")
	}
	// Output hash must be non-empty (at least 32 hex chars = 16 bytes).
	if len(outputHash) < 32 {
		return types.ErrEmptyOutputHash.Wrap("output_hash must be at least 32 hex characters")
	}

	// Mark as COMPLETED (not SUCCESS) — escrow stays locked during challenge window.
	inv.Status = types.StatusCompleted
	inv.OutputHash = outputHash
	inv.UsageReport = usageReport
	inv.CompletedHeight = ctx.BlockHeight()
	if err := k.setInvocation(ctx, inv); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"invocation_completed",
		sdk.NewAttribute("invocation_id", invocationID),
		sdk.NewAttribute("capability_id", inv.CapabilityId),
		sdk.NewAttribute("output_hash", outputHash),
		sdk.NewAttribute("challenge_window_ends", fmt.Sprintf("%d", ctx.BlockHeight()+types.ChallengeWindow)),
	))

	return nil
}

// ClaimInvocation allows the provider to claim payment after the challenge window passes.
func (k Keeper) ClaimInvocation(ctx sdk.Context, invocationID, caller string) error {
	inv, err := k.GetInvocation(ctx, invocationID)
	if err != nil {
		return err
	}
	if inv.Status != types.StatusCompleted {
		return types.ErrInvalidStatus.Wrapf("invocation %s is not in COMPLETED status (status: %s)", invocationID, inv.Status)
	}
	if inv.Provider != caller {
		return types.ErrUnauthorized.Wrap("only the provider can claim an invocation")
	}
	// Challenge window must have passed.
	if ctx.BlockHeight() < inv.CompletedHeight+types.ChallengeWindow {
		return types.ErrChallengeWindow.Wrapf(
			"challenge window ends at block %d, current block %d",
			inv.CompletedHeight+types.ChallengeWindow, ctx.BlockHeight(),
		)
	}

	// Release escrow to pay the provider.
	if inv.Amount.IsPositive() && inv.EscrowId != "" {
		if err := k.settlementKeeper.ReleaseEscrow(ctx, inv.EscrowId, caller); err != nil {
			return err
		}
	}

	inv.Status = types.StatusSuccess
	if err := k.setInvocation(ctx, inv); err != nil {
		return err
	}

	// Update capability stats.
	cap, err := k.GetCapability(ctx, inv.CapabilityId)
	if err == nil {
		cap.TotalCalls++
		cap.TotalEarned = cap.TotalEarned.Add(inv.Amount.Amount)
		if cap.TotalCalls == 1 {
			cap.SuccessRate = 10000
		} else {
			cap.SuccessRate = (cap.SuccessRate*(uint32(cap.TotalCalls)-1) + 10000) / uint32(cap.TotalCalls)
		}
		if err := k.setCapability(ctx, cap); err != nil {
			ctx.EventManager().EmitEvent(sdk.NewEvent("capability_stats_update_failed",
				sdk.NewAttribute("capability_id", inv.CapabilityId),
				sdk.NewAttribute("error", err.Error()),
			))
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"invocation_claimed",
		sdk.NewAttribute("invocation_id", invocationID),
		sdk.NewAttribute("capability_id", inv.CapabilityId),
		sdk.NewAttribute("provider", inv.Provider),
	))

	return nil
}

// DisputeInvocation allows the consumer to dispute a completed invocation within the challenge window.
func (k Keeper) DisputeInvocation(ctx sdk.Context, invocationID, caller, reason string) error {
	inv, err := k.GetInvocation(ctx, invocationID)
	if err != nil {
		return err
	}
	if inv.Status != types.StatusCompleted {
		return types.ErrInvalidStatus.Wrapf("invocation %s is not in COMPLETED status (status: %s)", invocationID, inv.Status)
	}
	if inv.Consumer != caller {
		return types.ErrUnauthorized.Wrap("only the consumer can dispute an invocation")
	}
	// Must be within the challenge window.
	if ctx.BlockHeight() >= inv.CompletedHeight+types.ChallengeWindow {
		return types.ErrChallengeWindow.Wrapf(
			"challenge window ended at block %d, current block %d",
			inv.CompletedHeight+types.ChallengeWindow, ctx.BlockHeight(),
		)
	}

	// Refund escrow to consumer.
	if inv.Amount.IsPositive() && inv.EscrowId != "" {
		if err := k.settlementKeeper.RefundEscrow(ctx, inv.EscrowId, caller); err != nil {
			return err
		}
	}

	inv.Status = types.StatusDisputed
	if err := k.setInvocation(ctx, inv); err != nil {
		return err
	}

	// Update capability stats (record dispute as failure).
	cap, err := k.GetCapability(ctx, inv.CapabilityId)
	if err == nil {
		cap.TotalCalls++
		if cap.TotalCalls == 1 {
			cap.SuccessRate = 0
		} else {
			cap.SuccessRate = (cap.SuccessRate * (uint32(cap.TotalCalls) - 1)) / uint32(cap.TotalCalls)
		}
		if err := k.setCapability(ctx, cap); err != nil {
			ctx.EventManager().EmitEvent(sdk.NewEvent("capability_stats_update_failed",
				sdk.NewAttribute("capability_id", inv.CapabilityId),
				sdk.NewAttribute("error", err.Error()),
			))
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"invocation_disputed",
		sdk.NewAttribute("invocation_id", invocationID),
		sdk.NewAttribute("capability_id", inv.CapabilityId),
		sdk.NewAttribute("consumer", inv.Consumer),
		sdk.NewAttribute("reason", reason),
	))

	return nil
}

// FailInvocation marks an invocation as failed and refunds the escrow.
func (k Keeper) FailInvocation(ctx sdk.Context, invocationID, caller string) error {
	inv, err := k.GetInvocation(ctx, invocationID)
	if err != nil {
		return err
	}
	if inv.Status != types.StatusPending {
		return types.ErrInvalidStatus.Wrapf("invocation %s is not pending (status: %s)", invocationID, inv.Status)
	}
	// Either the provider or consumer can report a failure.
	if inv.Provider != caller && inv.Consumer != caller {
		return types.ErrUnauthorized.Wrap("only the provider or consumer can fail an invocation")
	}

	// Refund escrow to consumer.
	if inv.Amount.IsPositive() && inv.EscrowId != "" {
		if err := k.settlementKeeper.RefundEscrow(ctx, inv.EscrowId, caller); err != nil {
			return err
		}
	}

	inv.Status = types.StatusFailed
	if err := k.setInvocation(ctx, inv); err != nil {
		return err
	}

	// Update capability stats (record failure).
	cap, err := k.GetCapability(ctx, inv.CapabilityId)
	if err == nil {
		cap.TotalCalls++
		// successRate = (oldRate * (totalCalls-1) + 0) / totalCalls
		if cap.TotalCalls == 1 {
			cap.SuccessRate = 0
		} else {
			cap.SuccessRate = (cap.SuccessRate * (uint32(cap.TotalCalls) - 1)) / uint32(cap.TotalCalls)
		}
		if err := k.setCapability(ctx, cap); err != nil {
			ctx.EventManager().EmitEvent(sdk.NewEvent("capability_stats_update_failed",
				sdk.NewAttribute("capability_id", inv.CapabilityId),
				sdk.NewAttribute("error", err.Error()),
			))
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"invocation_failed",
		sdk.NewAttribute("invocation_id", invocationID),
		sdk.NewAttribute("capability_id", inv.CapabilityId),
	))

	return nil
}

// UpdateCapability updates mutable fields of a capability. Only the owner can update.
func (k Keeper) UpdateCapability(ctx sdk.Context, msg *types.MsgUpdateCapability) error {
	cap, err := k.GetCapability(ctx, msg.CapabilityId)
	if err != nil {
		return err
	}
	if cap.Provider != msg.Creator {
		return types.ErrUnauthorized.Wrap("only the provider can update the capability")
	}

	if msg.EndpointUrl != "" {
		cap.EndpointUrl = msg.EndpointUrl
	}
	if msg.PricePerCall != nil && msg.PricePerCall.IsValid() {
		cap.PricePerCall = *msg.PricePerCall
	}
	if msg.RateLimit > 0 {
		params := k.GetParams(ctx)
		if params.MaxRateLimit > 0 && msg.RateLimit > params.MaxRateLimit {
			return types.ErrRateLimitExceeded.Wrapf("requested %d exceeds max %d", msg.RateLimit, params.MaxRateLimit)
		}
		cap.RateLimit = msg.RateLimit
	}
	if msg.Description != "" {
		cap.Description = msg.Description
	}

	if err := k.setCapability(ctx, cap); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"capability_updated",
		sdk.NewAttribute("capability_id", msg.CapabilityId),
		sdk.NewAttribute("provider", msg.Creator),
	))

	return nil
}

// DeactivateCapability marks a capability as inactive. Only the owner can deactivate.
func (k Keeper) DeactivateCapability(ctx sdk.Context, msg *types.MsgDeactivateCapability) error {
	cap, err := k.GetCapability(ctx, msg.CapabilityId)
	if err != nil {
		return err
	}
	if cap.Provider != msg.Creator {
		return types.ErrUnauthorized.Wrap("only the provider can deactivate the capability")
	}

	cap.IsActive = false
	if err := k.setCapability(ctx, cap); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"capability_deactivated",
		sdk.NewAttribute("capability_id", msg.CapabilityId),
		sdk.NewAttribute("provider", msg.Creator),
	))

	return nil
}

// --- Invocation helpers ---

func (k Keeper) setInvocation(ctx sdk.Context, inv types.Invocation) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&inv)
	if err != nil {
		return err
	}
	store.Set(types.InvocationKey(inv.Id), bz)
	return nil
}

// GetInvocation returns an invocation by ID.
func (k Keeper) GetInvocation(ctx sdk.Context, id string) (types.Invocation, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.InvocationKey(id))
	if bz == nil {
		return types.Invocation{}, types.ErrInvocationNotFound.Wrapf("id: %s", id)
	}
	var inv types.Invocation
	if err := k.cdc.Unmarshal(bz, &inv); err != nil {
		return types.Invocation{}, err
	}
	return inv, nil
}

// --- Genesis ---

// InitGenesis initializes the module from genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	_ = k.SetParams(ctx, gs.Params)
	for _, cap := range gs.Capabilities {
		_ = k.setCapability(ctx, cap)
	}
	for _, inv := range gs.Invocations {
		_ = k.setInvocation(ctx, inv)
	}
}

// ExportGenesis exports the module genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	caps := k.ListCapabilities(ctx, "")

	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.InvocationKeyPrefix)
	defer iter.Close()

	var invocations []types.Invocation
	for ; iter.Valid(); iter.Next() {
		var inv types.Invocation
		if err := k.cdc.Unmarshal(iter.Value(), &inv); err != nil {
			continue
		}
		invocations = append(invocations, inv)
	}

	return &types.GenesisState{
		Params:       k.GetParams(ctx),
		Capabilities: caps,
		Invocations:  invocations,
	}
}

// containsTag checks if a slice of tags contains the given tag.
func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
