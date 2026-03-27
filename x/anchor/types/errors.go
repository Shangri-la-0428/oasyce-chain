package types

import "cosmossdk.io/errors"

var (
	ErrDuplicateAnchor  = errors.Register(ModuleName, 2, "trace already anchored")
	ErrInvalidTraceID   = errors.Register(ModuleName, 3, "invalid trace_id")
	ErrInvalidPubkey    = errors.Register(ModuleName, 4, "invalid node_pubkey")
	ErrInvalidSignature = errors.Register(ModuleName, 5, "invalid trace_signature")
	ErrInvalidSigner    = errors.Register(ModuleName, 6, "signer does not match pubkey")
	ErrInvalidAddress   = errors.Register(ModuleName, 7, "invalid address")
	ErrBatchTooLarge    = errors.Register(ModuleName, 8, "batch exceeds maximum of 50 anchors")
	ErrAnchorNotFound   = errors.Register(ModuleName, 9, "anchor not found")
	ErrInvalidCapability = errors.Register(ModuleName, 10, "invalid capability")
	ErrInvalidTimestamp  = errors.Register(ModuleName, 11, "invalid timestamp")
)
