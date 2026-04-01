package types

import "encoding/binary"

const (
	// ModuleName is the name of the onboarding module.
	ModuleName = "onboarding"

	// StoreKey is the store key string for onboarding.
	StoreKey = ModuleName

	// RouterKey is the message route for onboarding.
	RouterKey = ModuleName
)

// Key prefixes for the onboarding store.
var (
	RegistrationKeyPrefix = []byte{0x01}
	ParamsKey             = []byte{0x02}
	TotalRegistrationsKey = []byte{0x03}
	DeadlineIndexPrefix   = []byte{0x04} // deadline index: prefix + unix_seconds(8) + address
)

// RegistrationKey returns the store key for a registration by address.
func RegistrationKey(address string) []byte {
	return append(RegistrationKeyPrefix, []byte(address)...)
}

// DeadlineIndexKey returns the deadline index key: prefix + unix_seconds(8 BE) + address.
func DeadlineIndexKey(unixSeconds int64, address string) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(unixSeconds))
	key := make([]byte, 0, 1+8+len(address))
	key = append(key, DeadlineIndexPrefix...)
	key = append(key, bz...)
	key = append(key, []byte(address)...)
	return key
}

// DeadlineIndexEndKey returns the exclusive end key for scanning expired deadlines up to the given time.
func DeadlineIndexEndKey(unixSeconds int64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(unixSeconds+1))
	return append(DeadlineIndexPrefix, bz...)
}
