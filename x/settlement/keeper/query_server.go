package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/oasyce/chain/x/settlement/types"
)

// QueryServer implements the settlement query service.
type QueryServer struct {
	Keeper
}

// NewQueryServer returns an implementation of the settlement QueryServer.
func NewQueryServer(keeper Keeper) QueryServer {
	return QueryServer{Keeper: keeper}
}

// QueryEscrow returns a single escrow by ID.
func (q QueryServer) QueryEscrow(ctx sdk.Context, escrowID string) (*QueryEscrowResponse, error) {
	escrow, found := q.Keeper.GetEscrow(ctx, escrowID)
	if !found {
		return nil, types.ErrEscrowNotFound.Wrapf("escrow %s not found", escrowID)
	}
	return &QueryEscrowResponse{Escrow: escrow}, nil
}

// QueryEscrowsByCreator returns all escrows created by an address.
func (q QueryServer) QueryEscrowsByCreator(ctx sdk.Context, creator string) (*QueryEscrowsByCreatorResponse, error) {
	escrows := q.Keeper.GetEscrowsByCreator(ctx, creator)
	return &QueryEscrowsByCreatorResponse{Escrows: escrows}, nil
}

// QueryBondingCurvePrice returns the current bonding curve price for an asset.
func (q QueryServer) QueryBondingCurvePrice(ctx sdk.Context, assetID string) (*QueryBondingCurvePriceResponse, error) {
	price, err := q.Keeper.GetPrice(ctx, assetID)
	if err != nil {
		return nil, err
	}
	state, found := q.Keeper.GetBondingCurveState(ctx, assetID)
	if !found {
		return nil, types.ErrBondingCurveNotFound.Wrapf("asset %s", assetID)
	}
	return &QueryBondingCurvePriceResponse{
		CurrentPrice: sdk.NewCoin("uoas", price),
		State:        state,
	}, nil
}

// Response types for queries.

// QueryEscrowResponse is the response for QueryEscrow.
type QueryEscrowResponse struct {
	Escrow types.Escrow `json:"escrow"`
}

// QueryEscrowsByCreatorResponse is the response for QueryEscrowsByCreator.
type QueryEscrowsByCreatorResponse struct {
	Escrows []types.Escrow `json:"escrows"`
}

// QueryBondingCurvePriceResponse is the response for QueryBondingCurvePrice.
type QueryBondingCurvePriceResponse struct {
	CurrentPrice sdk.Coin                `json:"current_price"`
	State        types.BondingCurveState `json:"state"`
}
