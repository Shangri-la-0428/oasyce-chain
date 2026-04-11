package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/oasyce/chain/x/sigil/keeper"
	"github.com/oasyce/chain/x/sigil/types"
)

func TestQueryPulses_ActiveRemainingBlocks(t *testing.T) {
	k, ctx := setupKeeper(t)
	ctx = ctx.WithBlockHeight(500)
	require.NoError(t, k.SetParams(ctx, types.Params{
		DormantThreshold:  100,
		DissolveThreshold: 200,
		SubmitWindow:      10,
	}))

	require.NoError(t, k.SetSigil(ctx, types.Sigil{
		SigilId:          "SIG_active_query",
		Creator:          "oasyce1activequery",
		Status:           types.SigilStatusActive,
		LastActiveHeight: 300,
		DimensionPulses:  map[string]int64{"anchor": 450},
	}))
	k.SetActiveCount(ctx, 1)

	resp, err := keeper.NewQueryServer(k).Pulses(sdk.WrapSDKContext(ctx), &types.QueryPulsesRequest{
		SigilId: "SIG_active_query",
	})
	require.NoError(t, err)
	require.Equal(t, int64(450), resp.MaxPulseHeight)
	require.Equal(t, int64(50), resp.BlocksUntilDormant)
	require.Equal(t, int64(0), resp.BlocksUntilDissolve)
	require.Equal(t, int32(types.SigilStatusActive), resp.Status)
	require.Equal(t, int64(450), resp.DimensionPulses["anchor"])
}

func TestQueryPulses_DormantRemainingBlocks(t *testing.T) {
	k, ctx := setupKeeper(t)
	ctx = ctx.WithBlockHeight(500)
	require.NoError(t, k.SetParams(ctx, types.Params{
		DormantThreshold:  100,
		DissolveThreshold: 200,
		SubmitWindow:      10,
	}))

	require.NoError(t, k.SetSigil(ctx, types.Sigil{
		SigilId:          "SIG_dormant_query",
		Creator:          "oasyce1dormantquery",
		Status:           types.SigilStatusDormant,
		LastActiveHeight: 100,
		DimensionPulses:  map[string]int64{"anchor": 380},
	}))

	resp, err := keeper.NewQueryServer(k).Pulses(sdk.WrapSDKContext(ctx), &types.QueryPulsesRequest{
		SigilId: "SIG_dormant_query",
	})
	require.NoError(t, err)
	require.Equal(t, int64(380), resp.MaxPulseHeight)
	require.Equal(t, int64(0), resp.BlocksUntilDormant)
	require.Equal(t, int64(80), resp.BlocksUntilDissolve)
	require.Equal(t, int32(types.SigilStatusDormant), resp.Status)
}

func TestQueryPulses_DissolvedReturnsZeroWindows(t *testing.T) {
	k, ctx := setupKeeper(t)
	ctx = ctx.WithBlockHeight(500)

	require.NoError(t, k.SetSigil(ctx, types.Sigil{
		SigilId:          "SIG_dissolved_query",
		Creator:          "oasyce1dissolvedquery",
		Status:           types.SigilStatusDissolved,
		LastActiveHeight: 120,
		DimensionPulses:  map[string]int64{"anchor": 200},
	}))

	resp, err := keeper.NewQueryServer(k).Pulses(sdk.WrapSDKContext(ctx), &types.QueryPulsesRequest{
		SigilId: "SIG_dissolved_query",
	})
	require.NoError(t, err)
	require.Equal(t, int64(200), resp.MaxPulseHeight)
	require.Equal(t, int64(0), resp.BlocksUntilDormant)
	require.Equal(t, int64(0), resp.BlocksUntilDissolve)
	require.Equal(t, int32(types.SigilStatusDissolved), resp.Status)
}
