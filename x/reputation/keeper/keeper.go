package keeper

import (
	"encoding/binary"
	"fmt"
	"math"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/reputation/types"
)

// DecayHalfLifeDays is the half-life for time decay in days.
const DecayHalfLifeDays = 30

// Keeper manages the reputation module's state.
type Keeper struct {
	cdc              codec.BinaryCodec
	storeKey         storetypes.StoreKey
	capabilityKeeper types.CapabilityKeeper
	authority        string
}

// NewKeeper creates a new reputation Keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	capabilityKeeper types.CapabilityKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		capabilityKeeper: capabilityKeeper,
		authority:        authority,
	}
}

// Authority returns the module authority address.
func (k Keeper) Authority() string {
	return k.authority
}

// ---------------------------------------------------------------------------
// Params
// ---------------------------------------------------------------------------

// GetParams returns the reputation module parameters.
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

// SetParams sets the reputation module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ParamsKey, bz)
	return nil
}

// ---------------------------------------------------------------------------
// Feedback CRUD
// ---------------------------------------------------------------------------

// GetFeedback retrieves a feedback by ID.
func (k Keeper) GetFeedback(ctx sdk.Context, feedbackID string) (types.Feedback, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.FeedbackKey(feedbackID))
	if bz == nil {
		return types.Feedback{}, false
	}
	var fb types.Feedback
	if err := k.cdc.Unmarshal(bz, &fb); err != nil {
		return types.Feedback{}, false
	}
	return fb, true
}

// SetFeedback persists a feedback to the store.
func (k Keeper) SetFeedback(ctx sdk.Context, fb types.Feedback) error {
	bz, err := k.cdc.Marshal(&fb)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.FeedbackKey(fb.Id), bz)
	return nil
}

// RebuildFeedbackIndex rebuilds secondary indexes for a feedback (used during InitGenesis).
func (k Keeper) RebuildFeedbackIndex(ctx sdk.Context, fb types.Feedback) {
	k.setFeedbackIndex(ctx, fb)
}

// setFeedbackIndex creates secondary index entries for a feedback.
func (k Keeper) setFeedbackIndex(ctx sdk.Context, fb types.Feedback) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.FeedbackByToKey(fb.To, fb.Id), []byte(fb.Id))
	store.Set(types.FeedbackByInvKey(fb.InvocationId, fb.From), []byte(fb.Id))
}

// hasFeedbackForInvocation checks if a feedback from a specific address already exists for an invocation.
func (k Keeper) hasFeedbackForInvocation(ctx sdk.Context, invocationID, from string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.FeedbackByInvKey(invocationID, from))
}

// generateFeedbackID creates a unique feedback ID.
func (k Keeper) generateFeedbackID(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.FeedbackCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(types.FeedbackCounterKey, newBz)
	return fmt.Sprintf("FB_%016x", counter)
}

// GetFeedbacksByTarget returns all feedbacks for a given target address.
func (k Keeper) GetFeedbacksByTarget(ctx sdk.Context, target string) []types.Feedback {
	store := ctx.KVStore(k.storeKey)
	prefix := types.FeedbackByToIteratorPrefix(target)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var feedbacks []types.Feedback
	for ; iter.Valid(); iter.Next() {
		feedbackID := string(iter.Value())
		fb, found := k.GetFeedback(ctx, feedbackID)
		if found {
			feedbacks = append(feedbacks, fb)
		}
	}
	return feedbacks
}

// ---------------------------------------------------------------------------
// Cooldown
// ---------------------------------------------------------------------------

// getCooldownTimestamp returns the last feedback timestamp from->to.
func (k Keeper) getCooldownTimestamp(ctx sdk.Context, from, to string) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CooldownKey(from, to))
	if bz == nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(bz))
}

// setCooldownTimestamp stores the last feedback timestamp from->to.
func (k Keeper) setCooldownTimestamp(ctx sdk.Context, from, to string, timestamp int64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(timestamp))
	store.Set(types.CooldownKey(from, to), bz)
}

// ---------------------------------------------------------------------------
// Reputation Score CRUD
// ---------------------------------------------------------------------------

// GetReputation returns the reputation score for an address.
func (k Keeper) GetReputation(ctx sdk.Context, address string) (types.ReputationScore, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ScoreKey(address))
	if bz == nil {
		return types.ReputationScore{}, false
	}
	var score types.ReputationScore
	if err := k.cdc.Unmarshal(bz, &score); err != nil {
		return types.ReputationScore{}, false
	}
	return score, true
}

// SetReputation persists a reputation score to the store.
func (k Keeper) SetReputation(ctx sdk.Context, score types.ReputationScore) error {
	bz, err := k.cdc.Marshal(&score)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ScoreKey(score.Address), bz)
	return nil
}

// ---------------------------------------------------------------------------
// Misbehavior Reports
// ---------------------------------------------------------------------------

// generateReportID creates a unique report ID.
func (k Keeper) generateReportID(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ReportCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(types.ReportCounterKey, newBz)
	return fmt.Sprintf("RPT_%016x", counter)
}

// SetReport persists a misbehavior report to the store.
func (k Keeper) SetReport(ctx sdk.Context, report types.MisbehaviorReport) error {
	bz, err := k.cdc.Marshal(&report)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ReportKey(report.Id), bz)
	return nil
}

// ---------------------------------------------------------------------------
// Business Logic
// ---------------------------------------------------------------------------

// SubmitFeedback validates and stores a new feedback, then updates the target's
// reputation score.
func (k Keeper) SubmitFeedback(ctx sdk.Context, creator, invocationID string, rating uint32, comment string) (string, error) {
	params := k.GetParams(ctx)

	// Validate rating range.
	if rating > params.MaxRating {
		return "", types.ErrInvalidRating.Wrapf("rating %d exceeds max %d", rating, params.MaxRating)
	}

	// Look up the invocation to determine the target (provider) and verify participation.
	inv, err := k.capabilityKeeper.GetInvocation(ctx, invocationID)
	if err != nil {
		return "", types.ErrInvocationNotFound.Wrapf("invocation %s: %s", invocationID, err)
	}

	// Determine the target: the feedback goes to the provider.
	target := inv.Provider

	// Prevent self-feedback.
	if creator == target {
		return "", types.ErrSelfFeedback
	}

	// Check for duplicate feedback (same invocation + same submitter).
	if k.hasFeedbackForInvocation(ctx, invocationID, creator) {
		return "", types.ErrDuplicateFeedback.Wrapf("already submitted feedback for invocation %s", invocationID)
	}

	// Check cooldown.
	now := ctx.BlockTime().Unix()
	lastFeedback := k.getCooldownTimestamp(ctx, creator, target)
	if lastFeedback > 0 && uint64(now-lastFeedback) < params.FeedbackCooldownSeconds {
		return "", types.ErrCooldownActive.Wrapf("cooldown expires in %d seconds",
			params.FeedbackCooldownSeconds-uint64(now-lastFeedback))
	}

	// Determine if feedback is verified (creator is the consumer of the invocation).
	verified := creator == inv.Consumer

	// Generate feedback ID and create the feedback.
	feedbackID := k.generateFeedbackID(ctx)
	fb := types.Feedback{
		Id:           feedbackID,
		InvocationId: invocationID,
		From:         creator,
		To:           target,
		Rating:       rating,
		Comment:      comment,
		Verified:     verified,
		Timestamp:    ctx.BlockTime(),
	}

	if err := k.SetFeedback(ctx, fb); err != nil {
		return "", err
	}
	k.setFeedbackIndex(ctx, fb)
	k.setCooldownTimestamp(ctx, creator, target, now)

	// Recalculate reputation score.
	if err := k.UpdateScore(ctx, target); err != nil {
		return "", err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"feedback_submitted",
		sdk.NewAttribute("feedback_id", feedbackID),
		sdk.NewAttribute("invocation_id", invocationID),
		sdk.NewAttribute("from", creator),
		sdk.NewAttribute("to", target),
		sdk.NewAttribute("rating", fmt.Sprintf("%d", rating)),
		sdk.NewAttribute("verified", fmt.Sprintf("%t", verified)),
	))

	return feedbackID, nil
}

// UpdateScore recalculates the reputation score for an address using
// time-decayed weighted average of all feedbacks.
//
// Score formula per the wire protocol spec:
//
//	decay = exp(-0.693 * age_days / 30)
//	weight = decay * verification_weight
//	trust_score = sum(weight_i * score_i) / sum(weight_i)
//
// where score_i = rating / 500.0 (normalized to 0.0-1.0)
func (k Keeper) UpdateScore(ctx sdk.Context, address string) error {
	params := k.GetParams(ctx)
	feedbacks := k.GetFeedbacksByTarget(ctx, address)

	now := ctx.BlockTime()

	if len(feedbacks) == 0 {
		score := types.ReputationScore{
			Address:           address,
			TotalScore:        0,
			TotalFeedbacks:    0,
			VerifiedFeedbacks: 0,
			LastUpdated:       now,
		}
		return k.SetReputation(ctx, score)
	}

	ln2 := 0.693147180559945
	nowUnix := now.Unix()

	var weightedSum float64
	var totalWeight float64
	var verifiedCount uint64

	for _, fb := range feedbacks {
		ageDays := float64(nowUnix-fb.Timestamp.Unix()) / 86400.0
		if ageDays < 0 {
			ageDays = 0
		}
		decay := math.Exp(-ln2 * ageDays / float64(DecayHalfLifeDays))

		var verificationWeight float64
		if fb.Verified {
			verificationWeight = params.VerifiedWeight.MustFloat64()
			verifiedCount++
		} else {
			verificationWeight = params.UnverifiedWeight.MustFloat64()
		}

		weight := decay * verificationWeight
		normalizedRating := float64(fb.Rating) / 500.0
		weightedSum += weight * normalizedRating
		totalWeight += weight
	}

	var finalScore float64
	if totalWeight > 0 {
		finalScore = weightedSum / totalWeight
	}

	// Scale to 0-500 range and store as uint64.
	scaledScore := uint64(math.Round(finalScore * 500.0))

	score := types.ReputationScore{
		Address:           address,
		TotalScore:        scaledScore,
		TotalFeedbacks:    uint64(len(feedbacks)),
		VerifiedFeedbacks: verifiedCount,
		LastUpdated:       now,
	}

	return k.SetReputation(ctx, score)
}

// ReportMisbehavior stores a misbehavior report for governance review.
func (k Keeper) ReportMisbehavior(ctx sdk.Context, creator, target, evidenceType string, evidence []byte) (string, error) {
	// Validate addresses.
	if _, err := sdk.AccAddressFromBech32(creator); err != nil {
		return "", types.ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(target); err != nil {
		return "", types.ErrInvalidAddress.Wrapf("invalid target: %s", err)
	}

	reportID := k.generateReportID(ctx)
	report := types.MisbehaviorReport{
		Id:           reportID,
		Creator:      creator,
		Target:       target,
		EvidenceType: evidenceType,
		Evidence:     evidence,
		Timestamp:    ctx.BlockTime().Unix(),
	}

	if err := k.SetReport(ctx, report); err != nil {
		return "", err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"misbehavior_reported",
		sdk.NewAttribute("report_id", reportID),
		sdk.NewAttribute("creator", creator),
		sdk.NewAttribute("target", target),
		sdk.NewAttribute("evidence_type", evidenceType),
	))

	return reportID, nil
}

// IterateAllScores iterates over all reputation scores and calls the callback.
// Returning true from the callback stops iteration.
func (k Keeper) IterateAllScores(ctx sdk.Context, cb func(score types.ReputationScore) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ScoreKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var score types.ReputationScore
		if err := k.cdc.Unmarshal(iter.Value(), &score); err != nil {
			continue
		}
		if cb(score) {
			break
		}
	}
}

// IterateAllFeedbacks iterates over all feedbacks and calls the callback.
func (k Keeper) IterateAllFeedbacks(ctx sdk.Context, cb func(fb types.Feedback) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.FeedbackKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var fb types.Feedback
		if err := k.cdc.Unmarshal(iter.Value(), &fb); err != nil {
			continue
		}
		if cb(fb) {
			break
		}
	}
}

// IterateAllReports iterates over all misbehavior reports and calls the callback.
func (k Keeper) IterateAllReports(ctx sdk.Context, cb func(report types.MisbehaviorReport) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ReportKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var report types.MisbehaviorReport
		if err := k.cdc.Unmarshal(iter.Value(), &report); err != nil {
			continue
		}
		if cb(report) {
			break
		}
	}
}
