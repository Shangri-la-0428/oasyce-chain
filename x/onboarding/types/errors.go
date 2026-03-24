package types

import "cosmossdk.io/errors"

var (
	ErrRegistrationNotFound = errors.Register(ModuleName, 2, "registration not found")
	ErrInvalidAddress       = errors.Register(ModuleName, 3, "invalid address")
	ErrInvalidParams        = errors.Register(ModuleName, 4, "invalid params")
	ErrInsufficientFunds    = errors.Register(ModuleName, 5, "insufficient funds")
	ErrAlreadyRegistered    = errors.Register(ModuleName, 6, "address already registered")
	ErrInvalidPoW           = errors.Register(ModuleName, 7, "invalid proof of work")
	ErrNotActive            = errors.Register(ModuleName, 8, "registration is not active")
	ErrDeadlineNotPassed    = errors.Register(ModuleName, 9, "repayment deadline has not passed")
)
