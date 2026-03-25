package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/bits"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/onboarding/types"
)

// Keeper manages the onboarding module's state.
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	bankKeeper types.BankKeeper
	authority  string
}

// NewKeeper creates a new onboarding Keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
		authority:  authority,
	}
}

// Authority returns the module authority address.
func (k Keeper) Authority() string {
	return k.authority
}

// ---------------------------------------------------------------------------
// Params CRUD
// ---------------------------------------------------------------------------

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

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
// Registration CRUD
// ---------------------------------------------------------------------------

func (k Keeper) GetRegistration(ctx sdk.Context, address string) (types.Registration, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.RegistrationKey(address))
	if bz == nil {
		return types.Registration{}, false
	}
	var reg types.Registration
	if err := k.cdc.Unmarshal(bz, &reg); err != nil {
		return types.Registration{}, false
	}
	return reg, true
}

func (k Keeper) SetRegistration(ctx sdk.Context, reg types.Registration) error {
	bz, err := k.cdc.Marshal(&reg)
	if err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(types.RegistrationKey(reg.Address), bz)
	return nil
}

func (k Keeper) IterateAllRegistrations(ctx sdk.Context, cb func(reg types.Registration) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.RegistrationKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var reg types.Registration
		if err := k.cdc.Unmarshal(iter.Value(), &reg); err != nil {
			continue
		}
		if cb(reg) {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// Total Registrations Counter (for halving epochs)
// ---------------------------------------------------------------------------

func (k Keeper) GetTotalRegistrations(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalRegistrationsKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetTotalRegistrations(ctx sdk.Context, count uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.TotalRegistrationsKey, bz)
}

func (k Keeper) IncrementTotalRegistrations(ctx sdk.Context) uint64 {
	count := k.GetTotalRegistrations(ctx) + 1
	k.SetTotalRegistrations(ctx, count)
	return count
}

// ---------------------------------------------------------------------------
// Halving Economics
// ---------------------------------------------------------------------------

// HalvingEpoch returns the current halving epoch based on total registrations.
//
//	Epoch 0: 0 – 10,000 registrations
//	Epoch 1: 10,001 – 50,000
//	Epoch 2: 50,001 – 200,000
//	Epoch 3: 200,001+
func HalvingEpoch(totalRegs uint64) uint32 {
	switch {
	case totalRegs <= 10_000:
		return 0
	case totalRegs <= 50_000:
		return 1
	case totalRegs <= 200_000:
		return 2
	default:
		return 3
	}
}

// HalvingAirdrop returns the airdrop amount in uoas for the given epoch.
// Base: 20 OAS, halves each epoch: 20 → 10 → 5 → 2.5
func HalvingAirdrop(epoch uint32) math.Int {
	base := math.NewInt(20_000_000) // 20 OAS in uoas
	for i := uint32(0); i < epoch; i++ {
		base = base.Quo(math.NewInt(2))
	}
	return base
}

// HalvingDifficulty returns the PoW difficulty for the given epoch.
// Base: 16 bits, +2 per epoch: 16 → 18 → 20 → 22
func HalvingDifficulty(epoch uint32) uint32 {
	return 16 + 2*epoch
}

// ---------------------------------------------------------------------------
// PoW Verification
// ---------------------------------------------------------------------------

// VerifyPoW checks that sha256(address || nonce_le_bytes) has at least
// `difficulty` leading zero bits.
func VerifyPoW(address string, nonce uint64, difficulty uint32) bool {
	data := make([]byte, len(address)+8)
	copy(data, address)
	binary.LittleEndian.PutUint64(data[len(address):], nonce)

	hash := sha256.Sum256(data)
	return LeadingZeroBits(hash[:]) >= int(difficulty)
}

// LeadingZeroBits counts the number of leading zero bits in a byte slice.
func LeadingZeroBits(b []byte) int {
	total := 0
	for _, v := range b {
		if v == 0 {
			total += 8
		} else {
			total += bits.LeadingZeros8(v)
			break
		}
	}
	return total
}

// ---------------------------------------------------------------------------
// Business Logic
// ---------------------------------------------------------------------------

// SelfRegister registers a new user via proof-of-work.
func (k Keeper) SelfRegister(ctx context.Context, msg types.MsgSelfRegister) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Check address not already registered.
	if _, found := k.GetRegistration(sdkCtx, msg.Creator); found {
		return math.Int{}, types.ErrAlreadyRegistered.Wrapf("address %s already registered", msg.Creator)
	}

	params := k.GetParams(sdkCtx)

	// Compute halving epoch from cumulative registration count.
	totalRegs := k.GetTotalRegistrations(sdkCtx)
	epoch := HalvingEpoch(totalRegs)
	effectiveDifficulty := HalvingDifficulty(epoch)
	effectiveAirdrop := HalvingAirdrop(epoch)

	// Use the stricter of params difficulty and halving difficulty.
	if params.PowDifficulty > effectiveDifficulty {
		effectiveDifficulty = params.PowDifficulty
	}

	// Verify proof-of-work with effective difficulty.
	if !VerifyPoW(msg.Creator, msg.Nonce, effectiveDifficulty) {
		return math.Int{}, types.ErrInvalidPoW.Wrapf(
			"sha256(%s || nonce) does not have %d leading zero bits (epoch %d)",
			msg.Creator, effectiveDifficulty, epoch,
		)
	}

	// Use the lesser of params airdrop and halving airdrop.
	airdropAmt := effectiveAirdrop
	if params.AirdropAmount.Amount.LT(airdropAmt) {
		airdropAmt = params.AirdropAmount.Amount
	}
	airdropCoins := sdk.NewCoins(sdk.NewCoin(params.AirdropAmount.Denom, airdropAmt))
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, airdropCoins); err != nil {
		return math.Int{}, err
	}
	userAddr, _ := sdk.AccAddressFromBech32(msg.Creator)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, userAddr, airdropCoins); err != nil {
		return math.Int{}, err
	}

	// Create registration record.
	reg := types.Registration{
		Address:       msg.Creator,
		AirdropAmount: airdropAmt,
		RepaidAmount:  math.ZeroInt(),
		Status:        types.REGISTRATION_STATUS_ACTIVE,
		RegisteredAt:  sdkCtx.BlockTime(),
		Deadline:      sdkCtx.BlockTime().Add(time.Duration(params.RepaymentDeadlineDays) * 24 * time.Hour),
		PowNonce:      msg.Nonce,
	}

	if err := k.SetRegistration(sdkCtx, reg); err != nil {
		return math.Int{}, err
	}

	// Increment total registrations counter (drives halving schedule).
	newTotal := k.IncrementTotalRegistrations(sdkCtx)

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"self_registered",
		sdk.NewAttribute("address", msg.Creator),
		sdk.NewAttribute("airdrop_amount", airdropAmt.String()),
		sdk.NewAttribute("epoch", fmt.Sprintf("%d", epoch)),
		sdk.NewAttribute("total_registrations", fmt.Sprintf("%d", newTotal)),
	))

	return airdropAmt, nil
}

// RepayDebt allows a user to repay their airdrop debt.
func (k Keeper) RepayDebt(ctx context.Context, msg types.MsgRepayDebt) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	reg, found := k.GetRegistration(sdkCtx, msg.Creator)
	if !found {
		return math.Int{}, types.ErrRegistrationNotFound.Wrapf("registration for %s not found", msg.Creator)
	}
	if reg.Status != types.REGISTRATION_STATUS_ACTIVE {
		return math.Int{}, types.ErrNotActive.Wrapf("registration for %s is not active", msg.Creator)
	}

	remaining := reg.AirdropAmount.Sub(reg.RepaidAmount)
	repayment := msg.Amount
	if repayment.GT(remaining) {
		repayment = remaining // Cap at remaining debt
	}

	params := k.GetParams(sdkCtx)

	// Transfer repayment from user to module, then burn (maintains supply).
	userAddr, _ := sdk.AccAddressFromBech32(msg.Creator)
	repayCoins := sdk.NewCoins(sdk.NewCoin(params.AirdropAmount.Denom, repayment))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, userAddr, types.ModuleName, repayCoins); err != nil {
		return math.Int{}, types.ErrInsufficientFunds.Wrapf("failed to repay: %s", err)
	}
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, repayCoins); err != nil {
		return math.Int{}, err
	}

	reg.RepaidAmount = reg.RepaidAmount.Add(repayment)
	newRemaining := reg.AirdropAmount.Sub(reg.RepaidAmount)

	if newRemaining.IsZero() {
		reg.Status = types.REGISTRATION_STATUS_REPAID
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"debt_repaid",
			sdk.NewAttribute("address", msg.Creator),
		))
	}

	if err := k.SetRegistration(sdkCtx, reg); err != nil {
		return math.Int{}, err
	}

	return newRemaining, nil
}
