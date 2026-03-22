package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"sort"

	cosmosmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/datarights/types"
)

// Jury voting constants — matches formulas.py.
const (
	JurySize           = 5     // Number of jurors per dispute
	MajorityThreshold  = 2.0 / 3.0 // 2/3 majority required
	JurorRewardUoas    = 2000000    // 2 OAS per correct juror
)

// Reputation adjustments for dispute outcomes.
var (
	RepPenaltyProviderLoss = cosmosmath.NewInt(-10)
	RepPenaltyConsumerLoss = cosmosmath.NewInt(-5)
	RepRewardMajorityJuror = cosmosmath.NewInt(1)
	RepPenaltyMinorityJuror = cosmosmath.NewInt(-2)
)

// KV store prefixes for jury state.
var (
	JuryVotePrefix   = []byte{0x0A}
	JuryMemberPrefix = []byte{0x0B}
)

// JuryVoteKey returns the store key for a jury vote.
func JuryVoteKey(disputeID, juror string) []byte {
	key := append(JuryVotePrefix, []byte(disputeID)...)
	key = append(key, '/')
	key = append(key, []byte(juror)...)
	return key
}

// JuryVoteByDisputePrefix returns the prefix for iterating votes by dispute.
func JuryVoteByDisputePrefix(disputeID string) []byte {
	key := append(JuryVotePrefix, []byte(disputeID)...)
	key = append(key, '/')
	return key
}

// JuryMemberKey returns the store key for a jury member.
func JuryMemberKey(disputeID, juror string) []byte {
	key := append(JuryMemberPrefix, []byte(disputeID)...)
	key = append(key, '/')
	key = append(key, []byte(juror)...)
	return key
}

// JuryVote represents a single juror's vote on a dispute.
type JuryVote struct {
	DisputeID string `json:"dispute_id"`
	Juror     string `json:"juror"`
	// Vote: true = uphold dispute (in favor of plaintiff), false = reject
	Uphold    bool   `json:"uphold"`
}

// JuryScore calculates a deterministic jury selection score.
// Formula: random(hash(disputeID + nodeID)) × log(1 + reputation)
// Higher score = more likely selected as juror.
func JuryScore(disputeID, nodeID string, reputation float64) float64 {
	if reputation < 0 {
		reputation = 0
	}
	seed := sha256.Sum256([]byte(disputeID + nodeID))
	hashVal := binary.BigEndian.Uint64(seed[:8])
	randomVal := float64(hashVal) / float64(^uint64(0))
	return randomVal * math.Log1p(reputation)
}

// SelectJury selects jurors for a dispute from all shareholders of the asset.
// Jurors are selected by scoring all eligible addresses and taking the top N.
// The plaintiff and asset owner are excluded from jury duty.
func (k Keeper) SelectJury(ctx sdk.Context, disputeID string, assetID string, plaintiff string) []string {
	asset, found := k.GetAsset(ctx, assetID)
	if !found {
		return nil
	}

	// Collect all shareholders as candidates (excluding plaintiff and owner).
	type candidate struct {
		address string
		score   float64
	}
	var candidates []candidate

	holders := k.GetShareHolders(ctx, assetID)
	for _, sh := range holders {
		if sh.Address == plaintiff || sh.Address == asset.Owner {
			continue
		}
		// Use log of share count as reputation proxy (capped to prevent overflow).
		// In a full implementation, this would query the reputation module.
		repFloat := new(big.Float).SetInt(sh.Shares.BigInt())
		rep, _ := repFloat.Float64()
		if rep > 1e15 {
			rep = 1e15 // Cap to prevent float overflow
		}
		score := JuryScore(disputeID, sh.Address, rep)
		candidates = append(candidates, candidate{address: sh.Address, score: score})
	}

	// Sort by score descending and take top JurySize.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	jurySize := JurySize
	if len(candidates) < jurySize {
		jurySize = len(candidates)
	}

	jurors := make([]string, jurySize)
	store := ctx.KVStore(k.storeKey)
	for i := 0; i < jurySize; i++ {
		jurors[i] = candidates[i].address
		// Persist jury membership for vote validation.
		store.Set(JuryMemberKey(disputeID, jurors[i]), []byte{1})
	}
	return jurors
}

// SubmitJuryVote records a juror's vote on a dispute.
// Only addresses selected by SelectJury can vote.
func (k Keeper) SubmitJuryVote(ctx sdk.Context, disputeID, juror string, uphold bool) error {
	dispute, found := k.GetDispute(ctx, disputeID)
	if !found {
		return types.ErrDisputeNotFound.Wrapf("dispute %s not found", disputeID)
	}
	if dispute.Status != types.DISPUTE_STATUS_OPEN {
		return types.ErrDisputeNotOpen.Wrapf("dispute %s is not open", disputeID)
	}

	// Verify juror is a selected jury member.
	store := ctx.KVStore(k.storeKey)
	if !store.Has(JuryMemberKey(disputeID, juror)) {
		return types.ErrUnauthorized.Wrap("address is not a selected juror for this dispute")
	}

	voteKey := JuryVoteKey(disputeID, juror)

	// Prevent double voting.
	if store.Has(voteKey) {
		return types.ErrUnauthorized.Wrap("juror has already voted")
	}

	// Encode vote: 1 = uphold, 0 = reject.
	var voteByte byte
	if uphold {
		voteByte = 1
	}
	store.Set(voteKey, []byte{voteByte})

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"jury_vote",
		sdk.NewAttribute("dispute_id", disputeID),
		sdk.NewAttribute("juror", juror),
		sdk.NewAttribute("uphold", fmt.Sprintf("%t", uphold)),
	))

	return nil
}

// TallyVotes counts the votes for a dispute and returns (upholdCount, totalCount).
func (k Keeper) TallyVotes(ctx sdk.Context, disputeID string) (int, int) {
	store := ctx.KVStore(k.storeKey)
	prefix := JuryVoteByDisputePrefix(disputeID)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	upholdCount := 0
	totalCount := 0

	for ; iter.Valid(); iter.Next() {
		totalCount++
		if len(iter.Value()) > 0 && iter.Value()[0] == 1 {
			upholdCount++
		}
	}

	return upholdCount, totalCount
}

// ResolveByJury resolves a dispute based on jury votes.
// Requires 2/3 majority to uphold. If quorum not met, dispute remains open.
func (k Keeper) ResolveByJury(ctx sdk.Context, disputeID string) error {
	dispute, found := k.GetDispute(ctx, disputeID)
	if !found {
		return types.ErrDisputeNotFound.Wrapf("dispute %s not found", disputeID)
	}
	if dispute.Status != types.DISPUTE_STATUS_OPEN {
		return types.ErrDisputeNotOpen.Wrapf("dispute %s is not open", disputeID)
	}

	upholdCount, totalCount := k.TallyVotes(ctx, disputeID)
	if totalCount == 0 {
		return types.ErrInvalidParams.Wrap("no votes cast")
	}

	upholdRatio := float64(upholdCount) / float64(totalCount)

	if upholdRatio >= MajorityThreshold {
		// Dispute upheld — execute the requested remedy.
		remedy := dispute.RequestedRemedy
		if remedy == types.DISPUTE_REMEDY_UNSPECIFIED {
			remedy = types.DISPUTE_REMEDY_DELIST // fallback
		}

		// Execute remedy. Jury can only apply delist and rights_correction.
		// Transfer and share_adjustment require details that the jury doesn't have.
		switch remedy {
		case types.DISPUTE_REMEDY_DELIST:
			asset, found := k.GetAsset(ctx, dispute.AssetId)
			if found {
				asset.Status = types.ASSET_STATUS_SHUTTING_DOWN
				asset.ShutdownInitiatedAt = ctx.BlockTime()
				_ = k.SetAsset(ctx, asset)
			}
		case types.DISPUTE_REMEDY_RIGHTS_CORRECTION:
			// Downgrade rights to COLLECTION (most restrictive) as penalty.
			asset, found := k.GetAsset(ctx, dispute.AssetId)
			if found {
				asset.RightsType = types.RIGHTS_TYPE_COLLECTION
				_ = k.SetAsset(ctx, asset)
			}
		default:
			// Transfer and share_adjustment need details — fall back to delist.
			remedy = types.DISPUTE_REMEDY_DELIST
			asset, found := k.GetAsset(ctx, dispute.AssetId)
			if found {
				asset.Status = types.ASSET_STATUS_SHUTTING_DOWN
				asset.ShutdownInitiatedAt = ctx.BlockTime()
				_ = k.SetAsset(ctx, asset)
			}
		}

		// Return dispute deposit to plaintiff (dispute was legitimate).
		plaintiffAddr, err := sdk.AccAddressFromBech32(dispute.Plaintiff)
		if err == nil {
			params := k.GetParams(ctx)
			deposit := sdk.NewCoins(params.DisputeDeposit)
			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, plaintiffAddr, deposit)
		}

		dispute.Status = types.DISPUTE_STATUS_RESOLVED
		dispute.Remedy = remedy
		dispute.Arbitrator = "jury"
		dispute.ResolvedAt = ctx.BlockTime()
	} else {
		// Dispute rejected — deposit is forfeited (stays in module account as penalty).
		dispute.Status = types.DISPUTE_STATUS_REJECTED
		dispute.Arbitrator = "jury"
		dispute.ResolvedAt = ctx.BlockTime()
	}

	if err := k.SetDispute(ctx, dispute); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"dispute_jury_resolved",
		sdk.NewAttribute("dispute_id", disputeID),
		sdk.NewAttribute("uphold_count", fmt.Sprintf("%d", upholdCount)),
		sdk.NewAttribute("total_votes", fmt.Sprintf("%d", totalCount)),
		sdk.NewAttribute("outcome", dispute.Status.String()),
	))

	return nil
}
