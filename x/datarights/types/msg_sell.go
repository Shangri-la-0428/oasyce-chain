package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgSellShares sells shares back to the bonding curve.
type MsgSellShares struct {
	// Creator is the address of the seller.
	Creator string `json:"creator"`
	// AssetId is the ID of the data asset.
	AssetId string `json:"asset_id"`
	// Shares is the number of shares to sell.
	Shares math.Int `json:"shares"`
	// MinPayoutOut is the minimum payout expected (slippage protection).
	// Optional: nil or zero disables the check.
	MinPayoutOut *math.Int `json:"min_payout_out,omitempty"`
}

// MsgSellSharesResponse is the response for MsgSellShares.
type MsgSellSharesResponse struct {
	// Payout is the amount received after fees.
	Payout math.Int `json:"payout"`
}

// ValidateBasic performs basic validation on MsgSellShares.
func (m *MsgSellShares) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return ErrInvalidAddress.Wrapf("invalid creator: %s", err)
	}
	if m.AssetId == "" {
		return ErrInvalidParams.Wrap("asset_id must not be empty")
	}
	if m.Shares.IsNil() || m.Shares.IsZero() || m.Shares.IsNegative() {
		return ErrInsufficientFunds.Wrap("shares must be positive")
	}
	return nil
}

// GetSigners returns the expected signers for MsgSellShares.
func (m *MsgSellShares) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{addr}
}
