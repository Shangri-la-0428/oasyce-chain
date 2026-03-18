package types

import "cosmossdk.io/errors"

var (
	ErrInvalidRating      = errors.Register(ModuleName, 2, "invalid rating")
	ErrDuplicateFeedback  = errors.Register(ModuleName, 3, "duplicate feedback")
	ErrInvocationNotFound = errors.Register(ModuleName, 4, "invocation not found")
	ErrSelfFeedback       = errors.Register(ModuleName, 5, "cannot submit feedback for yourself")
	ErrCooldownActive     = errors.Register(ModuleName, 6, "feedback cooldown is active")
	ErrInvalidAddress     = errors.Register(ModuleName, 7, "invalid address")
	ErrInvalidParams      = errors.Register(ModuleName, 8, "invalid params")
)
