package types

import "cosmossdk.io/errors"

var (
	ErrSigilNotFound     = errors.Register(ModuleName, 2, "sigil not found")
	ErrSigilExists       = errors.Register(ModuleName, 3, "sigil already exists")
	ErrSigilDissolved    = errors.Register(ModuleName, 4, "sigil is dissolved")
	ErrSigilDormant      = errors.Register(ModuleName, 5, "sigil is dormant")
	ErrNotSigilOwner     = errors.Register(ModuleName, 6, "not sigil owner")
	ErrBondNotFound      = errors.Register(ModuleName, 7, "bond not found")
	ErrBondExists        = errors.Register(ModuleName, 8, "bond already exists")
	ErrSelfBond          = errors.Register(ModuleName, 9, "cannot bond with self")
	ErrInvalidLineage    = errors.Register(ModuleName, 10, "invalid lineage")
	ErrInvalidAddress    = errors.Register(ModuleName, 11, "invalid address")
	ErrInvalidSigilID    = errors.Register(ModuleName, 12, "invalid sigil ID")
	ErrInvalidPublicKey  = errors.Register(ModuleName, 13, "invalid public key")
	ErrInvalidStateRoot  = errors.Register(ModuleName, 14, "invalid state root")
	ErrInvalidBondID     = errors.Register(ModuleName, 15, "invalid bond ID")
	ErrInvalidForkMode   = errors.Register(ModuleName, 16, "invalid fork mode")
	ErrInvalidMergeMode  = errors.Register(ModuleName, 17, "invalid merge mode")
	ErrDuplicateSigil    = errors.Register(ModuleName, 18, "duplicate sigil in genesis")
	ErrInvalidPulse      = errors.Register(ModuleName, 19, "invalid pulse")
)
