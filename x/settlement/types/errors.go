package types

import "cosmossdk.io/errors"

var (
	ErrEscrowNotFound    = errors.Register(ModuleName, 2, "escrow not found")
	ErrInvalidStatus     = errors.Register(ModuleName, 3, "invalid escrow status")
	ErrUnauthorized      = errors.Register(ModuleName, 4, "unauthorized")
	ErrInsufficientFunds = errors.Register(ModuleName, 5, "insufficient funds")
	ErrEscrowExpired     = errors.Register(ModuleName, 6, "escrow expired")
	ErrInvalidAddress    = errors.Register(ModuleName, 7, "invalid address")
	ErrInvalidParams     = errors.Register(ModuleName, 8, "invalid params")
	ErrBondingCurveNotFound = errors.Register(ModuleName, 9, "bonding curve state not found")
	ErrInvalidAssetID    = errors.Register(ModuleName, 10, "invalid asset ID")
)
