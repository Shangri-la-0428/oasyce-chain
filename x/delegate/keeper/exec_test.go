package keeper

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/tx/signing"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	baseapptestutil "github.com/cosmos/cosmos-sdk/baseapp/testutil"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoregistry"

	delegatetypes "github.com/oasyce/chain/x/delegate/types"
)

type execTestAddressCodec struct{}

func (execTestAddressCodec) StringToBytes(text string) ([]byte, error) {
	return sdk.AccAddressFromBech32(text)
}

func (execTestAddressCodec) BytesToString(bz []byte) (string, error) {
	return sdk.AccAddress(bz).String(), nil
}

type storeBackedBankKeeper struct {
	storeKey storetypes.StoreKey
}

func newStoreBackedBankKeeper(storeKey storetypes.StoreKey) *storeBackedBankKeeper {
	return &storeBackedBankKeeper{storeKey: storeKey}
}

func (b *storeBackedBankKeeper) balanceKey(addr sdk.AccAddress, denom string) []byte {
	return []byte(addr.String() + "|" + denom)
}

func unwrapExecContext(ctx context.Context) sdk.Context {
	if sdkCtx, ok := ctx.(sdk.Context); ok {
		return sdkCtx
	}
	return sdk.UnwrapSDKContext(ctx)
}

func (b *storeBackedBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	sdkCtx := unwrapExecContext(ctx)
	store := sdkCtx.KVStore(b.storeKey)
	bz := store.Get(b.balanceKey(addr, denom))
	if bz == nil {
		return sdk.NewCoin(denom, math.ZeroInt())
	}

	amt, ok := math.NewIntFromString(string(bz))
	if !ok {
		panic("invalid balance amount in test store")
	}
	return sdk.NewCoin(denom, amt)
}

func (b *storeBackedBankKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins(b.GetBalance(ctx, addr, "uoas"))
}

func (b *storeBackedBankKeeper) SetBalance(ctx sdk.Context, addr sdk.AccAddress, coin sdk.Coin) {
	store := ctx.KVStore(b.storeKey)
	store.Set(b.balanceKey(addr, coin.Denom), []byte(coin.Amount.String()))
}

func (b *storeBackedBankKeeper) AddBalance(ctx sdk.Context, addr sdk.AccAddress, coin sdk.Coin) {
	cur := b.GetBalance(ctx, addr, coin.Denom)
	b.SetBalance(ctx, addr, sdk.NewCoin(coin.Denom, cur.Amount.Add(coin.Amount)))
}

func (b *storeBackedBankKeeper) SubBalance(ctx sdk.Context, addr sdk.AccAddress, coin sdk.Coin) {
	cur := b.GetBalance(ctx, addr, coin.Denom)
	b.SetBalance(ctx, addr, sdk.NewCoin(coin.Denom, cur.Amount.Sub(coin.Amount)))
}

type execTestRouter struct {
	bank *storeBackedBankKeeper
}

type nilExecRouter struct{}

func (nilExecRouter) Handler(_ sdk.Msg) baseapp.MsgServiceHandler         { return nil }
func (nilExecRouter) HandlerByTypeURL(_ string) baseapp.MsgServiceHandler { return nil }

func (r execTestRouter) Handler(msg sdk.Msg) baseapp.MsgServiceHandler {
	switch msg.(type) {
	case *baseapptestutil.MsgKeyValue:
		return r.handleKeyValue
	default:
		return nil
	}
}

func (r execTestRouter) HandlerByTypeURL(_ string) baseapp.MsgServiceHandler { return nil }

func (r execTestRouter) handleKeyValue(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
	kv, ok := msg.(*baseapptestutil.MsgKeyValue)
	if !ok {
		return nil, errors.New("unexpected message type")
	}
	signer := sdk.MustAccAddressFromBech32(kv.Signer)
	amount, ok := math.NewIntFromString(string(kv.Value))
	if !ok {
		return nil, errors.New("invalid amount")
	}
	coin := sdk.NewCoin("uoas", amount)

	switch string(kv.Key) {
	case "fail":
		return nil, errors.New("forced handler failure")
	case "credit":
		r.bank.AddBalance(ctx, signer, coin)
	default:
		r.bank.SubBalance(ctx, signer, coin)
	}

	return &sdk.Result{Data: kv.Key}, nil
}

func execTestAddr(seed byte) sdk.AccAddress {
	return sdk.AccAddress(bytes.Repeat([]byte{seed}, 20))
}

func setupExecKeeper(t *testing.T) (Keeper, sdk.Context, *storeBackedBankKeeper, execTestRouter) {
	t.Helper()

	delegateStoreKey := storetypes.NewKVStoreKey(delegatetypes.StoreKey)
	bankStoreKey := storetypes.NewKVStoreKey("delegate_exec_bank")
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	cms := store.NewCommitMultiStore(db, logger, metrics.NoOpMetrics{})
	cms.MountStoreWithDB(delegateStoreKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(bankStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	ctx := sdk.NewContext(cms, cmtproto.Header{Time: time.Now()}, false, logger)

	ir, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: protoregistry.GlobalFiles,
		SigningOptions: signing.Options{
			AddressCodec:          execTestAddressCodec{},
			ValidatorAddressCodec: execTestAddressCodec{},
		},
	})
	require.NoError(t, err)
	baseapptestutil.RegisterInterfaces(ir)
	cdc := codec.NewProtoCodec(ir)

	bank := newStoreBackedBankKeeper(bankStoreKey)
	router := execTestRouter{bank: bank}

	return NewKeeper(cdc, delegateStoreKey, bank, router, "authority"), ctx, bank, router
}

func setExecPolicyAndDelegate(
	t *testing.T,
	k Keeper,
	ctx sdk.Context,
	principal sdk.AccAddress,
	delegate sdk.AccAddress,
	allowed []string,
	perTx int64,
	window int64,
	maxMsgs int32,
) {
	t.Helper()

	policy := delegatetypes.DelegatePolicy{
		Principal:        principal.String(),
		PerTxLimit:       sdk.NewCoin("uoas", math.NewInt(perTx)),
		WindowLimit:      sdk.NewCoin("uoas", math.NewInt(window)),
		WindowSeconds:    3600,
		AllowedMsgs:      allowed,
		CreatedAtSeconds: ctx.BlockTime().Unix(),
		MaxMsgsPerExec:   maxMsgs,
	}
	require.NoError(t, k.SetPolicy(ctx, policy))
	require.NoError(t, k.SetDelegate(ctx, delegatetypes.DelegateRecord{
		Delegate:  delegate.String(),
		Principal: principal.String(),
		Label:     "test",
	}))
}

func newExecMsg(signer sdk.AccAddress, op string, amount int64) *baseapptestutil.MsgKeyValue {
	return &baseapptestutil.MsgKeyValue{
		Key:    []byte(op),
		Value:  []byte(strconv.FormatInt(amount, 10)),
		Signer: signer.String(),
	}
}

func TestResolveAndAuthorize_DelegateNotFound(t *testing.T) {
	k, ctx, _, _ := setupExecKeeper(t)
	ec := &ExecContext{}

	err := k.resolveAndAuthorize(ctx, execTestAddr(2).String(), ec)
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrDelegateNotFound.Error())
}

func TestResolveAndAuthorize_PolicyNotFound(t *testing.T) {
	k, ctx, _, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	delegate := execTestAddr(2)
	require.NoError(t, k.SetDelegate(ctx, delegatetypes.DelegateRecord{
		Delegate:  delegate.String(),
		Principal: principal.String(),
	}))

	ec := &ExecContext{}
	err := k.resolveAndAuthorize(ctx, delegate.String(), ec)
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrPolicyNotFound.Error())
}

func TestResolveAndAuthorize_PolicyExpired(t *testing.T) {
	k, ctx, _, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	delegate := execTestAddr(2)

	require.NoError(t, k.SetPolicy(ctx, delegatetypes.DelegatePolicy{
		Principal:         principal.String(),
		PerTxLimit:        sdk.NewCoin("uoas", math.NewInt(10)),
		WindowLimit:       sdk.NewCoin("uoas", math.NewInt(100)),
		WindowSeconds:     3600,
		AllowedMsgs:       []string{sdk.MsgTypeURL(newExecMsg(principal, "debit", 1))},
		CreatedAtSeconds:  ctx.BlockTime().Add(-2 * time.Hour).Unix(),
		ExpirationSeconds: 60,
	}))
	require.NoError(t, k.SetDelegate(ctx, delegatetypes.DelegateRecord{
		Delegate:  delegate.String(),
		Principal: principal.String(),
	}))

	ec := &ExecContext{}
	err := k.resolveAndAuthorize(ctx, delegate.String(), ec)
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrPolicyExpired.Error())
}

func TestValidateInnerMsgs_MsgNotAllowed(t *testing.T) {
	k, _, _, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	ec := &ExecContext{
		Delegate: delegatetypes.DelegateRecord{Principal: principal.String()},
		Policy: delegatetypes.DelegatePolicy{
			AllowedMsgs: []string{sdk.MsgTypeURL(newExecMsg(principal, "debit", 1))},
		},
		AllowedMsgs: map[string]bool{
			sdk.MsgTypeURL(newExecMsg(principal, "debit", 1)): true,
		},
		InnerMsgs: []sdk.Msg{
			&baseapptestutil.MsgCounter{Counter: 1, Signer: principal.String()},
		},
	}

	err := k.validateInnerMsgs(sdk.Context{}, ec)
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrMsgNotAllowed.Error())
}

func TestValidateInnerMsgs_SignerMismatch(t *testing.T) {
	k, _, _, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	wrongSigner := execTestAddr(9)
	msg := newExecMsg(wrongSigner, "debit", 1)
	ec := &ExecContext{
		Delegate: delegatetypes.DelegateRecord{Principal: principal.String()},
		Policy: delegatetypes.DelegatePolicy{
			MaxMsgsPerExec: 1,
		},
		AllowedMsgs: map[string]bool{
			sdk.MsgTypeURL(msg): true,
		},
		InnerMsgs: []sdk.Msg{msg},
	}

	err := k.validateInnerMsgs(sdk.Context{}, ec)
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrSignerMismatch.Error())
}

func TestValidateInnerMsgs_TooManyMessages(t *testing.T) {
	k, _, _, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	msg1 := newExecMsg(principal, "debit", 1)
	msg2 := newExecMsg(principal, "credit", 1)
	ec := &ExecContext{
		Delegate: delegatetypes.DelegateRecord{Principal: principal.String()},
		Policy: delegatetypes.DelegatePolicy{
			MaxMsgsPerExec: 1,
		},
		AllowedMsgs: map[string]bool{
			sdk.MsgTypeURL(msg1): true,
			sdk.MsgTypeURL(msg2): true,
		},
		InnerMsgs: []sdk.Msg{msg1, msg2},
	}

	err := k.validateInnerMsgs(sdk.Context{}, ec)
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrTooManyMessages.Error())
}

func TestExecuteAndTrack_HappyPath(t *testing.T) {
	k, ctx, bank, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	bank.SetBalance(ctx, principal, sdk.NewCoin("uoas", math.NewInt(10)))

	cacheCtx, write := ctx.CacheContext()
	ec := &ExecContext{
		Policy:        delegatetypes.DelegatePolicy{PerTxLimit: sdk.NewCoin("uoas", math.NewInt(10))},
		PrincipalAddr: principal,
		InnerMsgs:     []sdk.Msg{newExecMsg(principal, "debit", 4)},
	}

	require.NoError(t, k.executeAndTrack(cacheCtx, ec))
	require.Equal(t, math.NewInt(4), ec.GrossOutflow)
	require.Len(t, ec.Results, 1)
	require.Equal(t, math.NewInt(10), bank.GetBalance(ctx, principal, "uoas").Amount)

	write()
	require.Equal(t, math.NewInt(6), bank.GetBalance(ctx, principal, "uoas").Amount)
}

func TestExecuteAndTrack_HandlerMissing(t *testing.T) {
	k, ctx, bank, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	bank.SetBalance(ctx, principal, sdk.NewCoin("uoas", math.NewInt(10)))
	k.router = nilExecRouter{}

	cacheCtx, _ := ctx.CacheContext()
	ec := &ExecContext{
		Policy:        delegatetypes.DelegatePolicy{PerTxLimit: sdk.NewCoin("uoas", math.NewInt(10))},
		PrincipalAddr: principal,
		InnerMsgs:     []sdk.Msg{newExecMsg(principal, "debit", 1)},
	}

	err := k.executeAndTrack(cacheCtx, ec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no handler")
}

func TestExecuteAndTrack_PartialFailureRollsBack(t *testing.T) {
	k, ctx, bank, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	bank.SetBalance(ctx, principal, sdk.NewCoin("uoas", math.NewInt(10)))

	cacheCtx, _ := ctx.CacheContext()
	ec := &ExecContext{
		Policy:        delegatetypes.DelegatePolicy{PerTxLimit: sdk.NewCoin("uoas", math.NewInt(20))},
		PrincipalAddr: principal,
		InnerMsgs: []sdk.Msg{
			newExecMsg(principal, "debit", 3),
			newExecMsg(principal, "fail", 2),
		},
	}

	err := k.executeAndTrack(cacheCtx, ec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "forced handler failure")
	require.Equal(t, math.NewInt(10), bank.GetBalance(ctx, principal, "uoas").Amount)
}

func TestEnforceSpendLimits_PerTxLimit(t *testing.T) {
	k, ctx, _, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	cacheCtx, write := ctx.CacheContext()
	ec := &ExecContext{
		Delegate: delegatetypes.DelegateRecord{Principal: principal.String()},
		Policy: delegatetypes.DelegatePolicy{
			PerTxLimit:    sdk.NewCoin("uoas", math.NewInt(5)),
			WindowLimit:   sdk.NewCoin("uoas", math.NewInt(20)),
			WindowSeconds: 3600,
		},
		GrossOutflow: math.NewInt(6),
	}

	err := k.enforceSpendLimits(cacheCtx, ec, write)
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrExceedsPerTxLimit.Error())
}

func TestEnforceSpendLimits_WindowLimit(t *testing.T) {
	k, ctx, _, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	require.NoError(t, k.SetSpendWindow(ctx, delegatetypes.SpendWindow{
		Principal:   principal.String(),
		WindowStart: ctx.BlockTime().Unix(),
		Spent:       sdk.NewCoin("uoas", math.NewInt(8)),
	}))

	cacheCtx, write := ctx.CacheContext()
	ec := &ExecContext{
		Delegate: delegatetypes.DelegateRecord{Principal: principal.String()},
		Policy: delegatetypes.DelegatePolicy{
			PerTxLimit:    sdk.NewCoin("uoas", math.NewInt(10)),
			WindowLimit:   sdk.NewCoin("uoas", math.NewInt(10)),
			WindowSeconds: 3600,
		},
		GrossOutflow: math.NewInt(3),
	}

	err := k.enforceSpendLimits(cacheCtx, ec, write)
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrExceedsWindowLimit.Error())
}

func TestEnforceSpendLimits_WindowReset(t *testing.T) {
	k, ctx, _, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	require.NoError(t, k.SetSpendWindow(ctx, delegatetypes.SpendWindow{
		Principal:   principal.String(),
		WindowStart: ctx.BlockTime().Add(-2 * time.Hour).Unix(),
		Spent:       sdk.NewCoin("uoas", math.NewInt(9)),
	}))

	futureCtx := ctx.WithBlockTime(ctx.BlockTime().Add(2 * time.Hour))
	cacheCtx, write := futureCtx.CacheContext()
	ec := &ExecContext{
		Delegate: delegatetypes.DelegateRecord{Principal: principal.String()},
		Policy: delegatetypes.DelegatePolicy{
			PerTxLimit:    sdk.NewCoin("uoas", math.NewInt(10)),
			WindowLimit:   sdk.NewCoin("uoas", math.NewInt(10)),
			WindowSeconds: 3600,
		},
		GrossOutflow: math.NewInt(4),
	}

	require.NoError(t, k.enforceSpendLimits(cacheCtx, ec, write))
	window, found := k.GetSpendWindow(futureCtx, principal.String())
	require.True(t, found)
	require.Equal(t, math.NewInt(4), window.Spent.Amount)
	require.Equal(t, futureCtx.BlockTime().Unix(), window.WindowStart)
}

func TestExecDelegate_MultipleDelegatesShareBudget(t *testing.T) {
	k, ctx, bank, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	delegateA := execTestAddr(2)
	delegateB := execTestAddr(3)
	allowed := []string{sdk.MsgTypeURL(newExecMsg(principal, "debit", 1))}
	setExecPolicyAndDelegate(t, k, ctx, principal, delegateA, allowed, 10, 10, 0)
	require.NoError(t, k.SetDelegate(ctx, delegatetypes.DelegateRecord{
		Delegate:  delegateB.String(),
		Principal: principal.String(),
		Label:     "b",
	}))
	bank.SetBalance(ctx, principal, sdk.NewCoin("uoas", math.NewInt(20)))

	_, err := k.ExecDelegate(ctx, delegateA.String(), []sdk.Msg{newExecMsg(principal, "debit", 7)})
	require.NoError(t, err)

	_, err = k.ExecDelegate(ctx, delegateB.String(), []sdk.Msg{newExecMsg(principal, "debit", 5)})
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrExceedsWindowLimit.Error())

	window, found := k.GetSpendWindow(ctx, principal.String())
	require.True(t, found)
	require.Equal(t, math.NewInt(7), window.Spent.Amount)
	require.Equal(t, math.NewInt(13), bank.GetBalance(ctx, principal, "uoas").Amount)
}

func TestExecDelegate_GrossOutflowDoesNotNetAgainstCredits(t *testing.T) {
	k, ctx, bank, _ := setupExecKeeper(t)
	principal := execTestAddr(1)
	delegate := execTestAddr(2)
	allowed := []string{sdk.MsgTypeURL(newExecMsg(principal, "debit", 1))}
	setExecPolicyAndDelegate(t, k, ctx, principal, delegate, allowed, 5, 20, 0)
	bank.SetBalance(ctx, principal, sdk.NewCoin("uoas", math.NewInt(10)))

	_, err := k.ExecDelegate(ctx, delegate.String(), []sdk.Msg{
		newExecMsg(principal, "debit", 8),
		newExecMsg(principal, "credit", 8),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), delegatetypes.ErrExceedsPerTxLimit.Error())
	require.Equal(t, math.NewInt(10), bank.GetBalance(ctx, principal, "uoas").Amount)
	_, found := k.GetSpendWindow(ctx, principal.String())
	require.False(t, found)
}
