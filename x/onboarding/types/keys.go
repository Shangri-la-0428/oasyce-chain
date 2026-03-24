package types

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
	RegistrationKeyPrefix  = []byte{0x01}
	ParamsKey              = []byte{0x02}
	TotalRegistrationsKey  = []byte{0x03}
)

// RegistrationKey returns the store key for a registration by address.
func RegistrationKey(address string) []byte {
	return append(RegistrationKeyPrefix, []byte(address)...)
}
