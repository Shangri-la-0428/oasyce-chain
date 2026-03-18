package types

const (
	// ModuleName defines the module name. Uses "oasyce_capability" to avoid
	// conflict with the Cosmos SDK built-in capability module.
	ModuleName = "oasyce_capability"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key.
	RouterKey = ModuleName
)

// Store key prefixes.
var (
	CapabilityKeyPrefix  = []byte{0x01}
	InvocationKeyPrefix  = []byte{0x02}
	ParamsKey            = []byte{0x03}
	CapByProviderPrefix  = []byte{0x04}
	InvocationCounterKey = []byte{0x05}
	CapabilityCounterKey = []byte{0x06}
)

// CapabilityKey returns the store key for a capability by ID.
func CapabilityKey(id string) []byte {
	return append(CapabilityKeyPrefix, []byte(id)...)
}

// InvocationKey returns the store key for an invocation by ID.
func InvocationKey(id string) []byte {
	return append(InvocationKeyPrefix, []byte(id)...)
}

// CapByProviderKey returns the store key prefix for capabilities by provider.
func CapByProviderKey(provider string) []byte {
	return append(CapByProviderPrefix, []byte(provider+"/")...)
}

// CapByProviderCapKey returns the store key for a specific capability under a provider index.
func CapByProviderCapKey(provider, capID string) []byte {
	return append(CapByProviderKey(provider), []byte(capID)...)
}
