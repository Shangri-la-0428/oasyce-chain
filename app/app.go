package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	gogoproto "github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	txsigning "cosmossdk.io/x/tx/signing"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/server/api"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	// IBC
	ibccapabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	ibccapabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibcporttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	transfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	transferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

	// Oasyce custom modules
	"github.com/oasyce/chain/docs"
	oasyceparams "github.com/oasyce/chain/app/params"
	capability "github.com/oasyce/chain/x/capability"
	capabilitykeeper "github.com/oasyce/chain/x/capability/keeper"
	capabilitytypes "github.com/oasyce/chain/x/capability/types"
	datarights "github.com/oasyce/chain/x/datarights"
	datarightskeeper "github.com/oasyce/chain/x/datarights/keeper"
	datarightstypes "github.com/oasyce/chain/x/datarights/types"
	reputation "github.com/oasyce/chain/x/reputation"
	reputationkeeper "github.com/oasyce/chain/x/reputation/keeper"
	reputationtypes "github.com/oasyce/chain/x/reputation/types"
	settlement "github.com/oasyce/chain/x/settlement"
	settlementkeeper "github.com/oasyce/chain/x/settlement/keeper"
	settlementtypes "github.com/oasyce/chain/x/settlement/types"
	work "github.com/oasyce/chain/x/work"
	workkeeper "github.com/oasyce/chain/x/work/keeper"
	worktypes "github.com/oasyce/chain/x/work/types"
	onboarding "github.com/oasyce/chain/x/onboarding"
	onboardingkeeper "github.com/oasyce/chain/x/onboarding/keeper"
	onboardingtypes "github.com/oasyce/chain/x/onboarding/types"
	halving "github.com/oasyce/chain/x/halving"
	halvingkeeper "github.com/oasyce/chain/x/halving/keeper"
	halvingtypes "github.com/oasyce/chain/x/halving/types"
	anchor "github.com/oasyce/chain/x/anchor"
	anchorkeeper "github.com/oasyce/chain/x/anchor/keeper"
	anchortypes "github.com/oasyce/chain/x/anchor/types"
)

const Name = "oasyce"

var (
	// DefaultNodeHome is the default home directory for the application daemon.
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager, used for setting up genesis
	// and other module utilities.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		consensus.AppModuleBasic{},
		vesting.AppModuleBasic{},
		// IBC modules
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		transfer.AppModuleBasic{},
		// Oasyce custom modules
		settlement.AppModuleBasic{},
		capability.AppModuleBasic{},
		reputation.AppModuleBasic{},
		datarights.AppModuleBasic{},
		work.AppModuleBasic{},
		onboarding.AppModuleBasic{},
		halving.AppModuleBasic{},
		anchor.AppModuleBasic{},
	)

	// Module account permissions.
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibcexported.ModuleName:         nil,
		transfertypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
		settlementtypes.ModuleName:     {authtypes.Burner},
		datarightstypes.ModuleName:     {authtypes.Burner},
		worktypes.ModuleName:           {authtypes.Burner},
		onboardingtypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
		halvingtypes.ModuleName:        {authtypes.Minter},
		capabilitytypes.ModuleName:     nil, // holds provider stakes
	}
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultNodeHome = filepath.Join(userHomeDir, ".oasyced")

	// Set the default bond denom so that all module DefaultGenesis calls
	// (staking, mint, gov, crisis) produce "uoas" instead of "stake".
	sdk.DefaultBondDenom = oasyceparams.BondDenom

	// Set address prefixes.
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(oasyceparams.AccountAddressPrefix, oasyceparams.AccountPubKeyPrefix)
	config.SetBech32PrefixForValidator(oasyceparams.ValidatorAddressPrefix, oasyceparams.ValidatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(oasyceparams.ConsNodeAddressPrefix, oasyceparams.ConsNodePubKeyPrefix)
	config.Seal()
}

// OasyceApp extends a Cosmos SDK application with the Oasyce-specific modules.
type OasyceApp struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// ---- Keepers ----

	// Standard Cosmos SDK keepers
	AccountKeeper   authkeeper.AccountKeeper
	BankKeeper      bankkeeper.Keeper
	StakingKeeper   *stakingkeeper.Keeper
	SlashingKeeper  slashingkeeper.Keeper
	MintKeeper      mintkeeper.Keeper
	DistrKeeper     distrkeeper.Keeper
	GovKeeper       *govkeeper.Keeper
	CrisisKeeper    *crisiskeeper.Keeper
	UpgradeKeeper   *upgradekeeper.Keeper
	ParamsKeeper    paramskeeper.Keeper
	EvidenceKeeper  evidencekeeper.Keeper
	FeeGrantKeeper  feegrantkeeper.Keeper
	AuthzKeeper     authzkeeper.Keeper
	ConsensusKeeper consensuskeeper.Keeper

	// IBC keepers
	IBCCapabilityKeeper *ibccapabilitykeeper.Keeper
	IBCKeeper           *ibckeeper.Keeper
	TransferKeeper      transferkeeper.Keeper
	ScopedIBCKeeper     ibccapabilitykeeper.ScopedKeeper
	ScopedTransferKeeper ibccapabilitykeeper.ScopedKeeper

	// Oasyce custom module keepers
	SettlementKeeper settlementkeeper.Keeper
	CapabilityKeeper capabilitykeeper.Keeper
	ReputationKeeper reputationkeeper.Keeper
	DataRightsKeeper datarightskeeper.Keeper
	WorkKeeper       workkeeper.Keeper
	OnboardingKeeper onboardingkeeper.Keeper
	HalvingKeeper    halvingkeeper.Keeper
	AnchorKeeper     anchorkeeper.Keeper

	// Module manager
	ModuleManager *module.Manager
	configurator  module.Configurator

	// store keys
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey
}

// NewOasyceApp returns a fully constructed OasyceApp.
func NewOasyceApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *OasyceApp {
	addrCodec := addresscodec.NewBech32Codec(oasyceparams.AccountAddressPrefix)
	valAddrCodec := addresscodec.NewBech32Codec(oasyceparams.ValidatorAddressPrefix)
	interfaceRegistry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: gogoproto.HybridResolver,
		SigningOptions: txsigning.Options{
			AddressCodec:          addrCodec,
			ValidatorAddressCodec: valAddrCodec,
		},
	})
	if err != nil {
		panic(err)
	}
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)

	std.RegisterLegacyAminoCodec(legacyAmino)
	std.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)

	bApp := baseapp.NewBaseApp(Name, logger, db, txConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		crisistypes.StoreKey,
		minttypes.StoreKey,
		distrtypes.StoreKey,
		slashingtypes.StoreKey,
		govtypes.StoreKey,
		paramstypes.StoreKey,
		upgradetypes.StoreKey,
		evidencetypes.StoreKey,
		feegrant.StoreKey,
		authzkeeper.StoreKey,
		consensustypes.StoreKey,
		// IBC store keys
		ibcexported.StoreKey,
		ibccapabilitytypes.StoreKey,
		transfertypes.StoreKey,
		// Oasyce custom module store keys
		settlementtypes.StoreKey,
		capabilitytypes.StoreKey,
		reputationtypes.StoreKey,
		datarightstypes.StoreKey,
		worktypes.StoreKey,
		onboardingtypes.StoreKey,
		anchortypes.StoreKey,
	)
	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys := storetypes.NewMemoryStoreKeys(ibccapabilitytypes.MemStoreKey)

	app := &OasyceApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		txConfig:          txConfig,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	// --- Init Params Keeper ---
	app.ParamsKeeper = initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])

	// --- Init Consensus Keeper ---
	app.ConsensusKeeper = consensuskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensustypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.ProvideEventService(),
	)
	bApp.SetParamStore(app.ConsensusKeeper.ParamsStore)

	// --- Init Account Keeper ---
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addresscodec.NewBech32Codec(oasyceparams.AccountAddressPrefix),
		oasyceparams.AccountAddressPrefix,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// --- Init Bank Keeper ---
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		BlockedAddresses(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		logger,
	)

	// --- Init Staking Keeper ---
	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		addresscodec.NewBech32Codec(oasyceparams.ValidatorAddressPrefix),
		addresscodec.NewBech32Codec(oasyceparams.ConsNodeAddressPrefix),
	)

	// --- Init Mint Keeper ---
	app.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[minttypes.StoreKey]),
		app.StakingKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// --- Init Distribution Keeper ---
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// --- Init Slashing Keeper ---
	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		app.StakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// --- Init Crisis Keeper ---
	app.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[crisistypes.StoreKey]),
		5, // invariant check period
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.AccountKeeper.AddressCodec(),
	)

	// --- Init Upgrade Keeper ---
	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		make(map[int64]bool),
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		DefaultNodeHome,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// --- Init Evidence Keeper ---
	app.EvidenceKeeper = *evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[evidencetypes.StoreKey]),
		app.StakingKeeper,
		app.SlashingKeeper,
		app.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)

	// --- Init FeeGrant Keeper ---
	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[feegrant.StoreKey]),
		app.AccountKeeper,
	)

	// --- Init Authz Keeper ---
	app.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[authzkeeper.StoreKey]),
		appCodec,
		app.MsgServiceRouter(),
		app.AccountKeeper,
	)

	// --- Init Gov Keeper ---
	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.DistrKeeper,
		app.MsgServiceRouter(),
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.GovKeeper = govKeeper

	// Register gov proposal handlers (legacy).
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper))
	govKeeper.SetLegacyRouter(govRouter)

	// Register staking hooks so slashing/distribution are notified of validator events.
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks()),
	)

	// --- Init IBC Capability Keeper ---
	app.IBCCapabilityKeeper = ibccapabilitykeeper.NewKeeper(
		appCodec,
		keys[ibccapabilitytypes.StoreKey],
		memKeys[ibccapabilitytypes.MemStoreKey],
	)

	// Create scoped keepers for IBC modules.
	scopedIBCKeeper := app.IBCCapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	scopedTransferKeeper := app.IBCCapabilityKeeper.ScopeToModule(transfertypes.ModuleName)
	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper

	// --- Init IBC Keeper ---
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
		app.StakingKeeper,
		app.UpgradeKeeper,
		scopedIBCKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// --- Init Transfer Keeper ---
	app.TransferKeeper = transferkeeper.NewKeeper(
		appCodec,
		keys[transfertypes.StoreKey],
		app.GetSubspace(transfertypes.ModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		scopedTransferKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Create IBC router and add transfer route.
	transferIBCModule := transfer.NewIBCModule(app.TransferKeeper)
	ibcRouter := ibcporttypes.NewRouter()
	ibcRouter.AddRoute(transfertypes.ModuleName, transferIBCModule)
	app.IBCKeeper.SetRouter(ibcRouter)

	// --- Oasyce Custom Module Keepers ---
	govAuthority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	app.SettlementKeeper = settlementkeeper.NewKeeper(
		appCodec,
		keys[settlementtypes.StoreKey],
		app.BankKeeper,
		govAuthority,
	)

	app.CapabilityKeeper = capabilitykeeper.NewKeeper(
		keys[capabilitytypes.StoreKey],
		appCodec,
		app.BankKeeper,
		app.SettlementKeeper,
		govAuthority,
	)

	app.ReputationKeeper = reputationkeeper.NewKeeper(
		appCodec,
		keys[reputationtypes.StoreKey],
		app.CapabilityKeeper,
		govAuthority,
	)

	app.DataRightsKeeper = datarightskeeper.NewKeeper(
		appCodec,
		keys[datarightstypes.StoreKey],
		app.BankKeeper,
		govAuthority,
	)
	app.SettlementKeeper.SetDatarightsKeeper(&app.DataRightsKeeper)

	app.WorkKeeper = workkeeper.NewKeeper(
		appCodec,
		keys[worktypes.StoreKey],
		app.BankKeeper,
		app.ReputationKeeper,
		govAuthority,
	)

	app.OnboardingKeeper = onboardingkeeper.NewKeeper(
		appCodec,
		keys[onboardingtypes.StoreKey],
		app.BankKeeper,
		govAuthority,
	)

	app.HalvingKeeper = halvingkeeper.NewKeeper(app.BankKeeper)

	app.AnchorKeeper = anchorkeeper.NewKeeper(
		appCodec,
		keys[anchortypes.StoreKey],
		govAuthority,
	)

	// --- Module Manager ---
	app.ModuleManager = module.NewManager(
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, txConfig),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		crisis.NewAppModule(app.CrisisKeeper, false, app.GetSubspace(crisistypes.ModuleName)),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		params.NewAppModule(app.ParamsKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		// IBC modules
		ibc.NewAppModule(app.IBCKeeper),
		ibctm.NewAppModule(),
		transfer.NewAppModule(app.TransferKeeper),
		// Oasyce custom modules
		settlement.NewAppModule(app.SettlementKeeper),
		capability.NewAppModule(appCodec, app.CapabilityKeeper),
		reputation.NewAppModule(app.ReputationKeeper),
		datarights.NewAppModule(appCodec, app.DataRightsKeeper),
		work.NewAppModule(appCodec, app.WorkKeeper),
		onboarding.NewAppModule(appCodec, app.OnboardingKeeper),
		halving.NewAppModule(app.HalvingKeeper),
		anchor.NewAppModule(appCodec, app.AnchorKeeper),
	)

	// Set order of module operations.
	app.ModuleManager.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		ibccapabilitytypes.ModuleName,
		minttypes.ModuleName,
		halvingtypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		ibcexported.ModuleName,
		transfertypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		vestingtypes.ModuleName,
		consensustypes.ModuleName,
		settlementtypes.ModuleName,
		capabilitytypes.ModuleName,
		reputationtypes.ModuleName,
		datarightstypes.ModuleName,
		worktypes.ModuleName,
		onboardingtypes.ModuleName,
		anchortypes.ModuleName,
	)

	app.ModuleManager.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		ibccapabilitytypes.ModuleName,
		transfertypes.ModuleName,
		feegrant.ModuleName,
		genutiltypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		minttypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		consensustypes.ModuleName,
		settlementtypes.ModuleName,
		capabilitytypes.ModuleName,
		reputationtypes.ModuleName,
		datarightstypes.ModuleName,
		worktypes.ModuleName,
		onboardingtypes.ModuleName,
		anchortypes.ModuleName,
		halvingtypes.ModuleName,
	)

	genesisModuleOrder := []string{
		ibccapabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		crisistypes.ModuleName,
		ibcexported.ModuleName,
		genutiltypes.ModuleName,
		transfertypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		consensustypes.ModuleName,
		settlementtypes.ModuleName,
		capabilitytypes.ModuleName,
		reputationtypes.ModuleName,
		datarightstypes.ModuleName,
		worktypes.ModuleName,
		onboardingtypes.ModuleName,
		anchortypes.ModuleName,
		halvingtypes.ModuleName,
	}
	app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)
	app.ModuleManager.SetOrderExportGenesis(genesisModuleOrder...)

	app.ModuleManager.RegisterInvariants(app.CrisisKeeper)

	// Register all module services (msg handlers + query servers).
	app.configurator = module.NewConfigurator(appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	if err := app.ModuleManager.RegisterServices(app.configurator); err != nil {
		panic(err)
	}

	// Register chain upgrade handlers (must be after RegisterServices).
	app.registerUpgradeHandlers()

	// Set ABCI handlers.
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// Mount stores.
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// Seal the IBC capability keeper after all scoped keepers are created.
	app.IBCCapabilityKeeper.Seal()

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(err)
		}
	}

	return app
}

// GetSubspace returns a param subspace for a given module name.
func (app *OasyceApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// LegacyAmino returns the app's legacy amino codec.
func (app *OasyceApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns the app's codec.
func (app *OasyceApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns the app's InterfaceRegistry.
func (app *OasyceApp) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns the app's TxConfig.
func (app *OasyceApp) TxConfig() client.TxConfig {
	return app.txConfig
}

// Configurator returns the module configurator (used by upgrade handlers).
func (app *OasyceApp) Configurator() module.Configurator {
	return app.configurator
}

// InitChainer handles the chain initialization from genesis.
func (app *OasyceApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState map[string]json.RawMessage
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	return app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
}

// BeginBlocker runs module begin-block logic.
func (app *OasyceApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.ModuleManager.BeginBlock(ctx)
}

// EndBlocker runs module end-block logic.
func (app *OasyceApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

// RegisterAPIRoutes registers all application module routes with the provided API server.
func (app *OasyceApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig serverconfig.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// --- AI Agent Discovery Endpoints ---

	// Serve llms.txt — the primary document for AI agents to understand the chain.
	apiSvr.Router.HandleFunc("/llms.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(docs.LLMSTxt)
	}).Methods("GET")

	// Serve OpenAPI specification.
	apiSvr.Router.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(docs.OpenAPISpec)
	}).Methods("GET")

	// Service discovery metadata for AI agents.
	apiSvr.Router.HandleFunc("/.well-known/oasyce.json", func(w http.ResponseWriter, r *http.Request) {
		chainID := ""
		if info, err := clientCtx.Client.Status(r.Context()); err == nil {
			chainID = info.NodeInfo.Network
		}
		discovery := map[string]interface{}{
			"name":        "Oasyce Agent Economy",
			"description": "On-chain property rights, service contracts, and arbitration for autonomous agents",
			"chain_id":    chainID,
			"version":     version.Version,
			"denom":       "uoas",
			"docs": map[string]string{
				"llms_txt": "/llms.txt",
				"openapi":  "/openapi.yaml",
			},
			"modules":    []string{"settlement", "capability", "datarights", "reputation", "work", "onboarding"},
			"onboarding": "oasyced util auto-register",
			"report_issue": map[string]string{
				"endpoint": "/api/v1/report-issue",
				"method":   "POST",
				"label":    "ai-reported",
			},
			"source": "https://github.com/Shangri-la-0428/oasyce-chain",
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		bz, _ := json.MarshalIndent(discovery, "", "  ")
		fmt.Fprintln(w, string(bz))
	}).Methods("GET")

	// Error codes catalog — machine-readable error recovery guide.
	apiSvr.Router.HandleFunc("/oasyce/v1/error-codes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(docs.ErrorCodes)
	}).Methods("GET")

	// --- Aggregate Query Endpoints ---
	app.registerAggregateEndpoints(apiSvr.Router)

	// Issue report proxy — agents POST here, node forwards to GitHub using D1ROSE bot token.
	// Token is read from OASYCE_REPORT_TOKEN env var (never in code).
	apiSvr.Router.HandleFunc("/api/v1/report-issue", func(w http.ResponseWriter, r *http.Request) {
		token := os.Getenv("OASYCE_REPORT_TOKEN")
		if token == "" {
			http.Error(w, `{"error":"issue reporting not configured on this node"}`, http.StatusServiceUnavailable)
			return
		}

		// Parse agent's request.
		var req struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.Title == "" {
			http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
			return
		}

		// Build GitHub API request.
		ghBody, _ := json.Marshal(map[string]interface{}{
			"title":  req.Title,
			"body":   req.Body,
			"labels": []string{"ai-reported"},
		})
		ghReq, _ := http.NewRequestWithContext(r.Context(), "POST",
			"https://api.github.com/repos/Shangri-la-0428/oasyce-chain/issues",
			bytes.NewReader(ghBody))
		ghReq.Header.Set("Authorization", "Bearer "+token)
		ghReq.Header.Set("Content-Type", "application/json")
		ghReq.Header.Set("Accept", "application/vnd.github+json")
		ghReq.Header.Set("User-Agent", "oasyce-node")

		resp, err := http.DefaultClient.Do(ghReq)
		if err != nil {
			http.Error(w, `{"error":"failed to reach GitHub"}`, http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Forward GitHub's response.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}).Methods("POST")
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *OasyceApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *OasyceApp) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(clientCtx, app.BaseApp.GRPCQueryRouter(), app.interfaceRegistry, app.Query)
}

// RegisterNodeService implements the Application.RegisterNodeService method.
func (app *OasyceApp) RegisterNodeService(clientCtx client.Context, cfg serverconfig.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}
	return modAccAddrs
}

// initParamsKeeper initializes the params keeper and its subspaces.
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)
	// IBC subspaces
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(transfertypes.ModuleName)

	return paramsKeeper
}
