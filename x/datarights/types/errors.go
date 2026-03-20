package types

import "cosmossdk.io/errors"

var (
	ErrAssetNotFound     = errors.Register(ModuleName, 2, "data asset not found")
	ErrDisputeNotFound   = errors.Register(ModuleName, 3, "dispute not found")
	ErrNotArbitrator     = errors.Register(ModuleName, 4, "caller is not the arbitrator")
	ErrAssetDelisted     = errors.Register(ModuleName, 5, "data asset is delisted")
	ErrInvalidCoCreators = errors.Register(ModuleName, 6, "invalid co-creators")
	ErrInvalidAddress    = errors.Register(ModuleName, 7, "invalid address")
	ErrInsufficientFunds = errors.Register(ModuleName, 8, "insufficient funds")
	ErrInvalidParams     = errors.Register(ModuleName, 9, "invalid params")
	ErrUnauthorized      = errors.Register(ModuleName, 10, "unauthorized")
	ErrInvalidRightsType = errors.Register(ModuleName, 11, "invalid rights type")
	ErrDisputeNotOpen    = errors.Register(ModuleName, 12, "dispute is not open")
	ErrDuplicateAsset        = errors.Register(ModuleName, 13, "duplicate content hash")
	ErrContentHashMismatch   = errors.Register(ModuleName, 14, "content hash mismatch")
	ErrSlippageExceeded      = errors.Register(ModuleName, 15, "slippage tolerance exceeded")
)
