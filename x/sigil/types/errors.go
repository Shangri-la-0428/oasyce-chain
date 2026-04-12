package types

import "cosmossdk.io/errors"

var (
	ErrSigilNotFound    = errors.Register(ModuleName, 2, "sigil not found")
	ErrSigilExists      = errors.Register(ModuleName, 3, "sigil already exists")
	ErrSigilDissolved   = errors.Register(ModuleName, 4, "sigil is dissolved")
	ErrNotSigilOwner    = errors.Register(ModuleName, 6, "not sigil owner")
	ErrBondNotFound     = errors.Register(ModuleName, 7, "bond not found")
	ErrBondExists       = errors.Register(ModuleName, 8, "bond already exists")
	ErrInvalidAddress   = errors.Register(ModuleName, 11, "invalid address")
	ErrInvalidSigilID   = errors.Register(ModuleName, 12, "invalid sigil ID")
	ErrInvalidPublicKey = errors.Register(ModuleName, 13, "invalid public key")
	ErrInvalidBondID    = errors.Register(ModuleName, 15, "invalid bond ID")
	ErrInvalidForkMode  = errors.Register(ModuleName, 16, "invalid fork mode")
	ErrInvalidMergeMode = errors.Register(ModuleName, 17, "invalid merge mode")
	ErrInvalidPulse     = errors.Register(ModuleName, 19, "invalid pulse")
	ErrInvalidParams    = errors.Register(ModuleName, 20, "invalid module parameters")
	ErrSigilNotActive   = errors.Register(ModuleName, 21, "sigil is not active")
)
