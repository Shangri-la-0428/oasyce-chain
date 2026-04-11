package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/sigil/types"
)

// BeginBlocker processes liveness decay using effective activity height
// (MaxPulseHeight):
// Active sigils with effective activity height <= (currentHeight - DormantThreshold)
// become Dormant.
// Dormant sigils with effective activity height <= (currentHeight - DissolveThreshold)
// become Dissolved.
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	currentHeight := ctx.BlockHeight()

	// Phase 1: Dissolve — range-scan the dormant liveness index for sigils
	// whose frozen effective activity height has passed the dissolve
	// threshold. The index is keyed by height so this is O(expiring), not
	// O(total dormant).
	dissolveThreshold := currentHeight - params.DissolveThreshold
	if dissolveThreshold > 0 {
		var staleDormantIDs []string
		k.IterateStaleDormantSigils(ctx, dissolveThreshold, func(sigilID string) bool {
			staleDormantIDs = append(staleDormantIDs, sigilID)
			return false
		})
		for _, sigilID := range staleDormantIDs {
			s, found := k.GetSigil(ctx, sigilID)
			if !found {
				continue
			}
			if types.SigilStatus(s.Status) != types.SigilStatusDormant {
				continue
			}

			s.Status = types.SigilStatusDissolved
			_ = k.SetSigil(ctx, s)

			ctx.EventManager().EmitEvent(sdk.NewEvent(
				"sigil_auto_dissolve",
				sdk.NewAttribute("sigil_id", s.SigilId),
				sdk.NewAttribute("height", fmt.Sprintf("%d", currentHeight)),
			))
		}
	}

	// Phase 2: Dormancy — scan for active sigils past the dormant threshold.
	dormantThreshold := currentHeight - params.DormantThreshold
	if dormantThreshold > 0 {
		var staleActiveIDs []string
		k.IterateStaleSigils(ctx, dormantThreshold, func(sigilID string) bool {
			staleActiveIDs = append(staleActiveIDs, sigilID)
			return false
		})
		for _, sigilID := range staleActiveIDs {
			s, found := k.GetSigil(ctx, sigilID)
			if !found {
				continue
			}
			if types.SigilStatus(s.Status) == types.SigilStatusActive {
				s.Status = types.SigilStatusDormant
				k.DecrementActiveCount(ctx)
				_ = k.SetSigil(ctx, s)

				ctx.EventManager().EmitEvent(sdk.NewEvent(
					"sigil_dormant",
					sdk.NewAttribute("sigil_id", s.SigilId),
					sdk.NewAttribute("height", fmt.Sprintf("%d", currentHeight)),
				))
			}
		}
	}

	return nil
}
