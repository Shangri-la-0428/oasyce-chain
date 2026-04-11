package app

import (
	"encoding/json"
	"net/http"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/gorilla/mux"

	datarightstypes "github.com/oasyce/chain/x/datarights/types"
	worktypes "github.com/oasyce/chain/x/work/types"
)

// queryCtx creates an sdk.Context for reading the latest committed state.
func (app *OasyceApp) queryCtx() (sdk.Context, error) {
	return app.CreateQueryContext(0, false)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// ---------- /oasyce/v1/agent-profile/{address} ----------

type agentProfileResponse struct {
	Address       string              `json:"address"`
	Balance       sdk.Coins           `json:"balance"`
	Reputation    *reputationInfo     `json:"reputation"`
	Capabilities  []capabilitySummary `json:"capabilities"`
	Earnings      *earningsInfo       `json:"earnings"`
	WorkTasks     *workInfo           `json:"work"`
	DataAssets    []dataAssetSummary  `json:"data_assets"`
	Shareholdings []shareholdingInfo  `json:"shareholdings"`
	Onboarding    *onboardingInfo     `json:"onboarding,omitempty"`
}

type reputationInfo struct {
	TotalScore     uint64 `json:"total_score"`
	TotalFeedbacks uint64 `json:"total_feedbacks"`
}

type capabilitySummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IsActive     bool   `json:"is_active"`
	TotalCalls   uint64 `json:"total_calls"`
	PricePerCall string `json:"price_per_call"`
}

type earningsInfo struct {
	TotalEarned sdk.Coins `json:"total_earned"`
	TotalCalls  uint64    `json:"total_calls"`
}

type workInfo struct {
	TasksCreated      int    `json:"tasks_created"`
	TasksExecuted     int    `json:"tasks_executed"`
	IsExecutor        bool   `json:"is_executor"`
	ExecutorCompleted uint64 `json:"executor_completed,omitempty"`
	ExecutorFailed    uint64 `json:"executor_failed,omitempty"`
}

type dataAssetSummary struct {
	AssetID     string `json:"asset_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	TotalShares string `json:"total_shares"`
}

type shareholdingInfo struct {
	AssetID string `json:"asset_id"`
	Shares  string `json:"shares"`
}

type onboardingInfo struct {
	Registered    bool   `json:"registered"`
	AirdropAmount string `json:"airdrop_amount,omitempty"`
	RepaidAmount  string `json:"repaid_amount,omitempty"`
	DebtRemaining string `json:"debt_remaining,omitempty"`
}

func (app *OasyceApp) handleAgentProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	addr := vars["address"]
	if addr == "" || !strings.HasPrefix(addr, "oasyce1") {
		writeError(w, http.StatusBadRequest, "invalid address")
		return
	}

	ctx, err := app.queryCtx()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "chain not ready")
		return
	}

	resp := agentProfileResponse{Address: addr}

	// Balance
	accAddr, err := sdk.AccAddressFromBech32(addr)
	if err == nil {
		resp.Balance = app.BankKeeper.GetAllBalances(ctx, accAddr)
	}
	if resp.Balance == nil {
		resp.Balance = sdk.Coins{}
	}

	// Reputation
	if score, found := app.ReputationKeeper.GetReputation(ctx, addr); found {
		resp.Reputation = &reputationInfo{
			TotalScore:     score.TotalScore,
			TotalFeedbacks: score.TotalFeedbacks,
		}
	} else {
		resp.Reputation = &reputationInfo{}
	}

	// Capabilities (as provider)
	caps := app.CapabilityKeeper.ListByProvider(ctx, addr)
	resp.Capabilities = make([]capabilitySummary, 0, len(caps))
	for _, c := range caps {
		resp.Capabilities = append(resp.Capabilities, capabilitySummary{
			ID:           c.Id,
			Name:         c.Name,
			IsActive:     c.IsActive,
			TotalCalls:   c.TotalCalls,
			PricePerCall: c.PricePerCall.String(),
		})
	}

	// Earnings (aggregate across all capabilities)
	var totalCalls uint64
	denomTotals := make(map[string]math.Int)
	for _, c := range caps {
		totalCalls += c.TotalCalls
		if c.TotalEarned.IsPositive() {
			denom := c.PricePerCall.Denom
			if denom == "" {
				denom = "uoas"
			}
			if existing, ok := denomTotals[denom]; ok {
				denomTotals[denom] = existing.Add(c.TotalEarned)
			} else {
				denomTotals[denom] = c.TotalEarned
			}
		}
	}
	var earnedCoins sdk.Coins
	for denom, amount := range denomTotals {
		earnedCoins = append(earnedCoins, sdk.NewCoin(denom, amount))
	}
	if len(earnedCoins) > 0 {
		earnedCoins = sdk.NewCoins(earnedCoins...)
	} else {
		earnedCoins = sdk.Coins{}
	}
	resp.Earnings = &earningsInfo{TotalEarned: earnedCoins, TotalCalls: totalCalls}

	// Work tasks
	wi := &workInfo{}
	app.WorkKeeper.IterateTasksByCreator(ctx, addr, func(_ worktypes.Task) bool {
		wi.TasksCreated++
		return false
	})
	app.WorkKeeper.IterateTasksByExecutor(ctx, addr, func(_ worktypes.Task) bool {
		wi.TasksExecuted++
		return false
	})
	if profile, found := app.WorkKeeper.GetExecutorProfile(ctx, addr); found {
		wi.IsExecutor = true
		wi.ExecutorCompleted = profile.TasksCompleted
		wi.ExecutorFailed = profile.TasksFailed
	}
	resp.WorkTasks = wi

	// Data assets (as owner)
	ownedAssets := app.DataRightsKeeper.ListAssetsByOwner(ctx, addr)
	resp.DataAssets = make([]dataAssetSummary, 0, len(ownedAssets))
	for _, a := range ownedAssets {
		resp.DataAssets = append(resp.DataAssets, dataAssetSummary{
			AssetID:     a.Id,
			Name:        a.Name,
			Status:      a.Status.String(),
			TotalShares: a.TotalShares.String(),
		})
	}

	// Shareholdings (across all assets)
	resp.Shareholdings = make([]shareholdingInfo, 0)
	app.DataRightsKeeper.IterateAllShareHolders(ctx, func(sh datarightstypes.ShareHolder) bool {
		if sh.Address == addr {
			resp.Shareholdings = append(resp.Shareholdings, shareholdingInfo{
				AssetID: sh.AssetId,
				Shares:  sh.Shares.String(),
			})
		}
		return false
	})

	// Onboarding
	if reg, found := app.OnboardingKeeper.GetRegistration(ctx, addr); found {
		debt := reg.AirdropAmount.Sub(reg.RepaidAmount)
		if debt.IsNegative() {
			debt = math.ZeroInt()
		}
		resp.Onboarding = &onboardingInfo{
			Registered:    true,
			AirdropAmount: reg.AirdropAmount.String(),
			RepaidAmount:  reg.RepaidAmount.String(),
			DebtRemaining: debt.String(),
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---------- /oasyce/v1/marketplace ----------

type marketplaceResponse struct {
	Stats        marketStats         `json:"stats"`
	Capabilities []capabilitySummary `json:"capabilities"`
	DataAssets   []dataAssetSummary  `json:"data_assets"`
	OpenTasks    []taskSummary       `json:"open_tasks"`
}

type marketStats struct {
	TotalCapabilities  int    `json:"total_capabilities"`
	TotalDataAssets    int    `json:"total_data_assets"`
	TotalOpenTasks     int    `json:"total_open_tasks"`
	TotalRegistrations uint64 `json:"total_registrations"`
}

type taskSummary struct {
	TaskID  uint64 `json:"task_id"`
	Type    string `json:"type"`
	Bounty  string `json:"bounty"`
	Status  string `json:"status"`
	Creator string `json:"creator"`
}

func (app *OasyceApp) handleMarketplace(w http.ResponseWriter, r *http.Request) {
	ctx, err := app.queryCtx()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "chain not ready")
		return
	}

	resp := marketplaceResponse{}

	// Active capabilities (limit 100)
	allCaps := app.CapabilityKeeper.ListCapabilities(ctx, "")
	resp.Capabilities = make([]capabilitySummary, 0)
	for _, c := range allCaps {
		if c.IsActive {
			resp.Capabilities = append(resp.Capabilities, capabilitySummary{
				ID:           c.Id,
				Name:         c.Name,
				IsActive:     true,
				TotalCalls:   c.TotalCalls,
				PricePerCall: c.PricePerCall.String(),
			})
			if len(resp.Capabilities) >= 100 {
				break
			}
		}
	}
	resp.Stats.TotalCapabilities = len(resp.Capabilities)

	// Active data assets (limit 100)
	resp.DataAssets = make([]dataAssetSummary, 0)
	app.DataRightsKeeper.IterateAllAssets(ctx, func(a datarightstypes.DataAsset) bool {
		if a.Status == datarightstypes.ASSET_STATUS_ACTIVE {
			resp.DataAssets = append(resp.DataAssets, dataAssetSummary{
				AssetID:     a.Id,
				Name:        a.Name,
				Status:      a.Status.String(),
				TotalShares: a.TotalShares.String(),
			})
		}
		return len(resp.DataAssets) >= 100
	})
	resp.Stats.TotalDataAssets = len(resp.DataAssets)

	// Open tasks (SUBMITTED status, limit 100)
	resp.OpenTasks = make([]taskSummary, 0)
	app.WorkKeeper.IterateTasksByStatus(ctx, worktypes.TASK_STATUS_SUBMITTED, func(t worktypes.Task) bool {
		resp.OpenTasks = append(resp.OpenTasks, taskSummary{
			TaskID:  t.Id,
			Type:    t.TaskType,
			Bounty:  t.Bounty.String(),
			Status:  t.Status.String(),
			Creator: t.Creator,
		})
		return len(resp.OpenTasks) >= 100
	})
	resp.Stats.TotalOpenTasks = len(resp.OpenTasks)

	// Total registrations
	resp.Stats.TotalRegistrations = app.OnboardingKeeper.GetTotalRegistrations(ctx)

	writeJSON(w, http.StatusOK, resp)
}

// ---------- /health ----------

type healthResponse struct {
	Status         string            `json:"status"`
	ChainID        string            `json:"chain_id"`
	Version        string            `json:"version"`
	BlockHeight    int64             `json:"block_height"`
	ModuleVersions map[string]uint64 `json:"module_versions"`
}

func visibleModuleVersions(vmap map[string]uint64) map[string]uint64 {
	visible := make(map[string]uint64)
	for _, mod := range []string{
		"settlement",
		"capability",
		"datarights",
		"reputation",
		"work",
		"onboarding",
		"halving",
		"anchor",
		"delegate",
		"sigil",
	} {
		if ver, ok := vmap[mod]; ok {
			visible[mod] = ver
		}
	}
	return visible
}

func (app *OasyceApp) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:  "ok",
		Version: version.Version,
	}

	ctx, err := app.queryCtx()
	if err != nil {
		resp.Status = "starting"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	resp.ChainID = ctx.ChainID()
	resp.BlockHeight = ctx.BlockHeight()

	// Module versions from upgrade keeper
	resp.ModuleVersions = make(map[string]uint64)
	vmap, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err == nil {
		resp.ModuleVersions = visibleModuleVersions(vmap)
	}

	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, http.StatusOK, resp)
}

// registerAggregateEndpoints is called from RegisterAPIRoutes.
func (app *OasyceApp) registerAggregateEndpoints(router *mux.Router) {
	router.HandleFunc("/oasyce/v1/agent-profile/{address}", app.handleAgentProfile).Methods("GET")
	router.HandleFunc("/oasyce/v1/marketplace", app.handleMarketplace).Methods("GET")
	router.HandleFunc("/health", app.handleHealth).Methods("GET")
}
