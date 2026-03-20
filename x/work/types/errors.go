package types

import "cosmossdk.io/errors"

var (
	ErrTaskNotFound         = errors.Register(ModuleName, 2, "task not found")
	ErrInvalidStatus        = errors.Register(ModuleName, 3, "invalid task status for this operation")
	ErrUnauthorized         = errors.Register(ModuleName, 4, "unauthorized: sender is not an assigned executor")
	ErrInvalidAddress       = errors.Register(ModuleName, 5, "invalid address")
	ErrInvalidParams        = errors.Register(ModuleName, 6, "invalid params")
	ErrInvalidBounty        = errors.Register(ModuleName, 7, "bounty must be positive and meet minimum")
	ErrInvalidTimeout       = errors.Register(ModuleName, 8, "timeout blocks out of allowed range")
	ErrInvalidRedundancy    = errors.Register(ModuleName, 9, "redundancy must be >= 1")
	ErrInvalidInputHash     = errors.Register(ModuleName, 10, "input hash must be 32 bytes (SHA-256)")
	ErrExecutorNotFound     = errors.Register(ModuleName, 11, "executor profile not found")
	ErrExecutorExists       = errors.Register(ModuleName, 12, "executor already registered")
	ErrExecutorInactive     = errors.Register(ModuleName, 13, "executor is not active")
	ErrCommitmentExists     = errors.Register(ModuleName, 14, "commitment already submitted for this task")
	ErrCommitmentNotFound   = errors.Register(ModuleName, 15, "commitment not found — must commit before reveal")
	ErrRevealMismatch       = errors.Register(ModuleName, 16, "reveal does not match commitment hash")
	ErrResultExists         = errors.Register(ModuleName, 17, "result already revealed for this task")
	ErrSelfAssignment       = errors.Register(ModuleName, 18, "task creator cannot be assigned as executor")
	ErrInsufficientReputation = errors.Register(ModuleName, 19, "executor reputation below minimum threshold")
	ErrDisputeBondTooLow    = errors.Register(ModuleName, 20, "dispute bond below minimum requirement")
	ErrNotEnoughExecutors   = errors.Register(ModuleName, 21, "not enough eligible executors for assignment")
	ErrTaskTypeNotSupported = errors.Register(ModuleName, 22, "executor does not support this task type")
	ErrInvalidTaskType      = errors.Register(ModuleName, 23, "task type cannot be empty")
)
