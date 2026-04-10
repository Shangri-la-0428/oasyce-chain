package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/sigil/types"
)

// BeginBlocker processes liveness decay:
// Active sigils with LastActiveHeight <= (currentHeight - DormantThreshold) become Dormant.
// Dormant sigils with LastActiveHeight <= (currentHeight - DissolveThreshold) become Dissolved.
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	currentHeight := ctx.BlockHeight()

	// Phase 1: Dissolve — scan for sigils that have been dormant long enough.
	dissolveThreshold := currentHeight - params.DissolveThreshold
	if dissolveThreshold > 0 {
		k.IterateStaleSigils(ctx, dissolveThreshold, func(sigilID string) bool {
			s, found := k.GetSigil(ctx, sigilID)
			if !found {
				return false
			}
			// Only dissolve dormant sigils (active ones get set to dormant first).
			if types.SigilStatus(s.Status) == types.SigilStatusDormant {
				k.DeleteSigilFromStatusIndex(ctx, types.SigilStatusDormant, s.SigilId)
				k.DeleteSigilFromLivenessIndex(ctx, s.LastActiveHeight, s.SigilId)
				s.Status = types.SigilStatusDissolved
				_ = k.SetSigil(ctx, s)

				ctx.EventManager().EmitEvent(sdk.NewEvent(
					"sigil_auto_dissolve",
					sdk.NewAttribute("sigil_id", s.SigilId),
					sdk.NewAttribute("height", fmt.Sprintf("%d", currentHeight)),
				))
			}
			return false
		})
	}

	// Phase 2: Dormancy — scan for active sigils past the dormant threshold.
	dormantThreshold := currentHeight - params.DormantThreshold
	if dormantThreshold > 0 {
		k.IterateStaleSigils(ctx, dormantThreshold, func(sigilID string) bool {
			s, found := k.GetSigil(ctx, sigilID)
			if !found {
				return false
			}
			if types.SigilStatus(s.Status) == types.SigilStatusActive {
				k.DeleteSigilFromStatusIndex(ctx, types.SigilStatusActive, s.SigilId)
				// Keep liveness index entry for dissolve phase scanning.
				s.Status = types.SigilStatusDormant
				k.DecrementActiveCount(ctx)
				_ = k.SetSigil(ctx, s)

				ctx.EventManager().EmitEvent(sdk.NewEvent(
					"sigil_dormant",
					sdk.NewAttribute("sigil_id", s.SigilId),
					sdk.NewAttribute("height", fmt.Sprintf("%d", currentHeight)),
				))
			}
			return false
		})
	}

	return nil
}

// MaxPulseHeight returns the most recent activity height across all dimensions.
// This is max(LastActiveHeight, max(DimensionPulses values)).
func MaxPulseHeight(s types.Sigil) int64 {
	h := s.LastActiveHeight
	for _, v := range s.DimensionPulses {
		if v > h {
			h = v
		}
	}
	return h
}
