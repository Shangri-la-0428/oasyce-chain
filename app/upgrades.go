package app

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// Upgrade plan names — each corresponds to a governance proposal.
const (
	UpgradeV053 = "v0.5.3"
	UpgradeV060 = "v0.6.0"
	UpgradeV070 = "v0.7.0"
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
