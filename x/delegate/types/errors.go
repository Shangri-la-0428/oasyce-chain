package types

import "cosmossdk.io/errors"

var (
	ErrPolicyNotFound     = errors.Register(ModuleName, 2, "delegation policy not found")
	ErrDelegateNotFound   = errors.Register(ModuleName, 3, "delegate not enrolled")
	ErrAlreadyEnrolled    = errors.Register(ModuleName, 4, "delegate already enrolled")
	ErrInvalidToken       = errors.Register(ModuleName, 5, "invalid enrollment token")
	ErrPolicyExpired      = errors.Register(ModuleName, 6, "delegation policy has expired")
	ErrMsgNotAllowed      = errors.Register(ModuleName, 7, "message type not allowed by policy")
	ErrExceedsPerTxLimit  = errors.Register(ModuleName, 8, "spend exceeds per-transaction limit")
	ErrExceedsWindowLimit = errors.Register(ModuleName, 9, "spend exceeds window limit")
	ErrInvalidAddress     = errors.Register(ModuleName, 10, "invalid address")
	ErrInvalidPolicy      = errors.Register(ModuleName, 11, "invalid policy parameters")
	ErrSelfDelegate       = errors.Register(ModuleName, 12, "cannot delegate to self")
	ErrSignerMismatch     = errors.Register(ModuleName, 13, "inner message signer must be the principal")
	ErrTooManyMessages    = errors.Register(ModuleName, 14, "too many messages in delegate execution")
)
