package types

import "encoding/binary"

const (
	ModuleName   = "work"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

var (
	TaskKeyPrefix          = []byte{0x01}
	CommitmentKeyPrefix    = []byte{0x02}
	ResultKeyPrefix        = []byte{0x03}
	StatusIndexPrefix      = []byte{0x04}
	ExecutorIndexPrefix    = []byte{0x05}
	CreatorIndexPrefix     = []byte{0x06}
	ExpiryIndexPrefix      = []byte{0x07}
	ParamsKey              = []byte{0x08}
	EpochStatsPrefix       = []byte{0x09}
	ExecutorProfilePrefix  = []byte{0x0A}
	TaskCounterKey         = []byte{0x0B}
	RevealExpiryIndexPrefix = []byte{0x0C}
)

// ---- Task keys ----

func TaskKey(taskID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	return append(TaskKeyPrefix, bz...)
}

// ---- Commitment keys ----

func CommitmentKey(taskID uint64, executor string) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	key := append(CommitmentKeyPrefix, bz...)
	key = append(key, []byte(executor)...)
	return key
}

func CommitmentIteratorPrefix(taskID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	return append(CommitmentKeyPrefix, bz...)
}

// ---- Result keys ----

func ResultKey(taskID uint64, executor string) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	key := append(ResultKeyPrefix, bz...)
	key = append(key, []byte(executor)...)
	return key
}

func ResultIteratorPrefix(taskID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	return append(ResultKeyPrefix, bz...)
}

// ---- Status index: prefix + status(1 byte) + taskID(8 bytes) ----

func StatusIndexKey(status TaskStatus, taskID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	key := append(StatusIndexPrefix, byte(status))
	key = append(key, bz...)
	return key
}

func StatusIndexIteratorPrefix(status TaskStatus) []byte {
	return append(StatusIndexPrefix, byte(status))
}

// ---- Executor index: prefix + executor + taskID ----

func ExecutorIndexKey(executor string, taskID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	key := append(ExecutorIndexPrefix, []byte(executor)...)
	key = append(key, '/')
	key = append(key, bz...)
	return key
}

func ExecutorIndexIteratorPrefix(executor string) []byte {
	key := append(ExecutorIndexPrefix, []byte(executor)...)
	key = append(key, '/')
	return key
}

// ---- Creator index: prefix + creator + taskID ----

func CreatorIndexKey(creator string, taskID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, taskID)
	key := append(CreatorIndexPrefix, []byte(creator)...)
	key = append(key, '/')
	key = append(key, bz...)
	return key
}

func CreatorIndexIteratorPrefix(creator string) []byte {
	key := append(CreatorIndexPrefix, []byte(creator)...)
	key = append(key, '/')
	return key
}

// ---- Expiry index: prefix + height(8 bytes) + taskID(8 bytes) ----

func ExpiryIndexKey(height, taskID uint64) []byte {
	key := make([]byte, 0, 1+8+8)
	key = append(key, ExpiryIndexPrefix...)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, height)
	key = append(key, bz...)
	binary.BigEndian.PutUint64(bz, taskID)
	key = append(key, bz...)
	return key
}

func RevealExpiryIndexKey(height, taskID uint64) []byte {
	key := make([]byte, 0, 1+8+8)
	key = append(key, RevealExpiryIndexPrefix...)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, height)
	key = append(key, bz...)
	binary.BigEndian.PutUint64(bz, taskID)
	key = append(key, bz...)
	return key
}

// ExpiryIndexEndKey returns the exclusive end key for scanning expired tasks up to (and including) the given height.
func ExpiryIndexEndKey(height uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, height+1)
	return append(ExpiryIndexPrefix, bz...)
}

func RevealExpiryIndexEndKey(height uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, height+1)
	return append(RevealExpiryIndexPrefix, bz...)
}

// ---- Executor profile key ----

func ExecutorProfileKey(addr string) []byte {
	return append(ExecutorProfilePrefix, []byte(addr)...)
}

// ---- Epoch stats key ----

func EpochStatsKey(epoch uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, epoch)
	return append(EpochStatsPrefix, bz...)
}
