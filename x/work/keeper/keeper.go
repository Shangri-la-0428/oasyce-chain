package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/work/types"
)

type Keeper struct {
	cdc              codec.BinaryCodec
	storeKey         storetypes.StoreKey
	bankKeeper       types.BankKeeper
	reputationKeeper types.ReputationKeeper
	authority        string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	reputationKeeper types.ReputationKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		bankKeeper:       bankKeeper,
		reputationKeeper: reputationKeeper,
		authority:        authority,
	}
}

// ---- Params ----

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
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

// ---- Task Counter ----

func (k Keeper) nextTaskID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TaskCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(types.TaskCounterKey, newBz)
	return counter
}

func (k Keeper) GetTaskCounter(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TaskCounterKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetTaskCounter(ctx sdk.Context, val uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, val)
	store.Set(types.TaskCounterKey, bz)
}
