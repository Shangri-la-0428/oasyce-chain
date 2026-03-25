package types

import "cosmossdk.io/errors"

var (
	ErrCapabilityNotFound = errors.Register(ModuleName, 2, "capability not found")
	ErrUnauthorized       = errors.Register(ModuleName, 3, "unauthorized")
	ErrInactive           = errors.Register(ModuleName, 4, "capability is inactive")
	ErrRateLimitExceeded  = errors.Register(ModuleName, 5, "rate limit exceeded")
	ErrInsufficientStake  = errors.Register(ModuleName, 6, "insufficient provider stake")
	ErrInvalidInput       = errors.Register(ModuleName, 7, "invalid input")
	ErrInvocationNotFound = errors.Register(ModuleName, 8, "invocation not found")
	ErrInvalidStatus      = errors.Register(ModuleName, 9, "invalid invocation status")
	ErrChallengeWindow    = errors.Register(ModuleName, 10, "challenge window violation")
	ErrEmptyOutputHash    = errors.Register(ModuleName, 11, "output hash cannot be empty")
)
