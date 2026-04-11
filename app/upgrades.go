package app

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	anchortypes "github.com/oasyce/chain/x/anchor/types"
	sigiltypes "github.com/oasyce/chain/x/sigil/types"
)

// Upgrade plan names — each corresponds to a governance proposal.
const (
	UpgradeV053 = "v0.5.3"
	UpgradeV060 = "v0.6.0"
	UpgradeV070 = "v0.7.0"
	UpgradeV080 = "v0.8.0"
)

// registerUpgradeHandlers registers all chain upgrade handlers.
// Called from NewOasyceApp after all keepers are initialized.
func (app *OasyceApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeV053,
		app.upgradeHandlerV053(),
	)
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeV060,
		app.upgradeHandlerV060(),
	)
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeV070,
		app.upgradeHandlerV070(),
	)
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeV080,
		app.upgradeHandlerV080(),
	)
}

// upgradeHandlerV053 handles the v0.5.2 → v0.5.3 upgrade.
//
// Changes in v0.5.3:
//   - x/anchor module added (new store key)
//   - AI agent docs updated (llms.txt, AGENTS.md)
//   - SDK v0.5.0 native signing support
func (app *OasyceApp) upgradeHandlerV053() func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		app.Logger().Info(
			fmt.Sprintf("applying upgrade %s at height %d — adding anchor module store", plan.Name, plan.Height),
		)
		return app.ModuleManager.RunMigrations(ctx, app.Configurator(), vm)
	}
}

// upgradeHandlerV070 handles the v0.6.0 → v0.7.0 upgrade.
//
// Changes in v0.7.0:
//   - x/sigil module added (new store key) — AI identity lifecycle
//   - x/onboarding → x/sigil cross-module integration (auto-creates Sigil on self-register)
//   - x/anchor sigil_id field + index (AnchorsBySigil query)
func (app *OasyceApp) upgradeHandlerV070() func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		app.Logger().Info(
			fmt.Sprintf("applying upgrade %s at height %d — adding sigil module store", plan.Name, plan.Height),
		)
		return app.ModuleManager.RunMigrations(ctx, app.Configurator(), vm)
	}
}

// upgradeHandlerV060 handles the v0.5.0 → v0.6.0 upgrade.
//
// Changes in v0.6.0:
//   - Aggregate query endpoints (agent-profile, marketplace, health) — no state migration needed
//   - Security boundary tests added — no state migration needed
//   - SellShares comment fix — no state migration needed
//   - llms.txt v3 — embedded doc update, no state migration
//
// This is a no-op upgrade that records the new module version map without
// modifying any on-chain state. It serves as a checkpoint to verify the
// upgrade governance flow works end-to-end before future upgrades that
// require real migrations.
func (app *OasyceApp) upgradeHandlerV060() func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		app.Logger().Info(
			fmt.Sprintf("applying upgrade %s at height %d", plan.Name, plan.Height),
		)

		// Run module migrations. This compares the stored ConsensusVersion
		// with each module's current ConsensusVersion() and runs any
		// registered migrations. For v0.6.0, all versions are unchanged,
		// so this is effectively a no-op that records the version map.
		return app.ModuleManager.RunMigrations(ctx, app.Configurator(), vm)
	}
}

// upgradeHandlerV080 handles the v0.7.x → v0.8.0 upgrade.
//
// Changes in v0.8.0:
//   - x/sigil v1 -> v2 state migration
//   - Rebuild active liveness index from effective activity height (MaxPulseHeight)
//   - No new module stores added
func (app *OasyceApp) upgradeHandlerV080() func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		rawSigilVersion, rawSigilPresent := vm[sigiltypes.ModuleName]
		current := app.ModuleManager.GetVersionMap()
		if vm == nil {
			vm = module.VersionMap{}
		}
		for mod, ver := range current {
			if _, ok := vm[mod]; !ok {
				vm[mod] = ver
			}
		}
		if !rawSigilPresent || rawSigilVersion == 0 {
			app.Logger().Info(
				fmt.Sprintf(
					"upgrade %s detected legacy sigil version-map state; treating existing sigil store as v1 before running v1 -> v2 migration",
					plan.Name,
				),
			)
			vm[sigiltypes.ModuleName] = 1
		}
		app.Logger().Info(
			fmt.Sprintf(
				"applying upgrade %s at height %d — sigil v1 -> v2 effective activity height migration",
				plan.Name,
				plan.Height,
			),
		)
		return app.ModuleManager.RunMigrations(ctx, app.Configurator(), vm)
	}
}

func storeUpgradesForPlan(name string) *storetypes.StoreUpgrades {
	switch name {
	case UpgradeV053:
		return &storetypes.StoreUpgrades{
			Added: []string{anchortypes.StoreKey},
		}
	case UpgradeV070:
		return &storetypes.StoreUpgrades{
			Added: []string{sigiltypes.StoreKey},
		}
	case UpgradeV080:
		// State migration only; no new stores are introduced.
		return nil
	default:
		return nil
	}
}
