package types

const (
	ModuleName = "delegate"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

// Store key prefixes.
var (
	PolicyKeyPrefix   = []byte{0x01} // 0x01 + principal_addr -> DelegatePolicy
	DelegateKeyPrefix = []byte{0x02} // 0x02 + delegate_addr -> DelegateRecord
	SpendKeyPrefix    = []byte{0x03} // 0x03 + principal_addr -> SpendWindow
	// Reverse index: principal -> list of delegate addresses
	PrincipalDelegatesPrefix = []byte{0x04} // 0x04 + principal_addr + delegate_addr -> []byte{}
)

func PolicyKey(principal string) []byte {
	return append(PolicyKeyPrefix, []byte(principal)...)
}

func DelegateKey(delegate string) []byte {
	return append(DelegateKeyPrefix, []byte(delegate)...)
}

func SpendKey(principal string) []byte {
	return append(SpendKeyPrefix, []byte(principal)...)
}

func PrincipalDelegateKey(principal, delegate string) []byte {
	key := make([]byte, 0, 1+len(principal)+1+len(delegate))
	key = append(key, PrincipalDelegatesPrefix...)
	key = append(key, []byte(principal)...)
	key = append(key, '/') // separator
	key = append(key, []byte(delegate)...)
	return key
}

func PrincipalDelegateIteratorKey(principal string) []byte {
	key := make([]byte, 0, 1+len(principal)+1)
	key = append(key, PrincipalDelegatesPrefix...)
	key = append(key, []byte(principal)...)
	key = append(key, '/')
	return key
}
