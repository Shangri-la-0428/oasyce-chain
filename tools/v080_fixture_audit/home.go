package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	chainapp "github.com/oasyce/chain/app"
	sigiltypes "github.com/oasyce/chain/x/sigil/types"
)

type appOptions map[string]interface{}

func (o appOptions) Get(key string) interface{} {
	return o[key]
}

func openAppAtHome(home string) (*chainapp.OasyceApp, func(), error) {
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, nil, err
	}
	dataDir := filepath.Join(absHome, "data")
	if _, err := os.Stat(dataDir); err != nil {
		return nil, nil, fmt.Errorf("fixture home missing data dir %s: %w", dataDir, err)
	}

	db, err := dbm.NewGoLevelDB("application", dataDir, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("open application db: %w", err)
	}

	oldHome := chainapp.DefaultNodeHome
	chainapp.DefaultNodeHome = absHome
	app := chainapp.NewOasyceApp(log.NewNopLogger(), db, nil, true, appOptions{})

	cleanup := func() {
		chainapp.DefaultNodeHome = oldHome
		_ = db.Close()
	}
	return app, cleanup, nil
}

func loadAuditFromHome(home string) (auditReport, error) {
	app, cleanup, err := openAppAtHome(home)
	if err != nil {
		return auditReport{}, err
	}
	defer cleanup()

	ctx := app.NewUncachedContext(false, cmtproto.Header{Height: maxInt64(1, app.LastBlockHeight())})
	vmap, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return auditReport{}, fmt.Errorf("load module version map: %w", err)
	}

	return collectAudit(ctx, app.SigilKeeper, app.SigilKeeper.StoreKey(), vmap[sigiltypes.ModuleName]), nil
}

func replayV080(sourceHome, workingHome string) (replayReport, error) {
	if sourceHome == "" {
		return replayReport{}, fmt.Errorf("source fixture home is required")
	}
	if err := copyDir(sourceHome, workingHome); err != nil {
		return replayReport{}, fmt.Errorf("copy fixture home: %w", err)
	}

	app, cleanup, err := openAppAtHome(workingHome)
	if err != nil {
		return replayReport{}, err
	}
	defer cleanup()

	ctx := app.NewUncachedContext(false, cmtproto.Header{Height: maxInt64(1, app.LastBlockHeight())})
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return replayReport{}, fmt.Errorf("load module version map: %w", err)
	}
	preVersion := vm[sigiltypes.ModuleName]
	if preVersion > 1 {
		return replayReport{}, fmt.Errorf("fixture sigil module version must be 0 or 1 before replay, got %d", preVersion)
	}

	before := collectAudit(ctx, app.SigilKeeper, app.SigilKeeper.StoreKey(), preVersion)

	plan := upgradetypes.Plan{
		Name:   chainapp.UpgradeV080,
		Height: ctx.BlockHeight() + 1,
		Info:   "sigil v1 -> v2 effective activity height migration",
	}
	if err := app.UpgradeKeeper.ScheduleUpgrade(ctx, plan); err != nil {
		return replayReport{}, fmt.Errorf("schedule upgrade: %w", err)
	}
	if err := app.UpgradeKeeper.ApplyUpgrade(ctx, plan); err != nil {
		return replayReport{}, fmt.Errorf("apply upgrade: %w", err)
	}

	updatedVM, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return replayReport{}, fmt.Errorf("reload module version map: %w", err)
	}
	after := collectAudit(ctx, app.SigilKeeper, app.SigilKeeper.StoreKey(), updatedVM[sigiltypes.ModuleName])
	if before.ActiveCount != after.ActiveCount {
		after.InvariantErrors = append(after.InvariantErrors,
			fmt.Sprintf("active_count changed across replay: before=%d after=%d", before.ActiveCount, after.ActiveCount),
		)
	}

	report := replayReport{
		SourceHome:  sourceHome,
		WorkingHome: workingHome,
		Before:      before,
		After:       after,
		Status:      "ok",
	}
	if len(after.InvariantErrors) > 0 || after.ModuleVersion != 2 {
		if after.ModuleVersion != 2 {
			report.After.InvariantErrors = append(report.After.InvariantErrors,
				fmt.Sprintf("sigil module version after replay = %d, want 2", after.ModuleVersion),
			)
		}
		report.Status = "failed"
	}
	return report, nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func copyDir(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case info.IsDir():
			return os.MkdirAll(target, info.Mode())
		case info.Mode().IsRegular():
			return copyFile(path, target, info.Mode())
		default:
			return nil
		}
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
