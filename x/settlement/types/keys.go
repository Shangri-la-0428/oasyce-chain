package types

const (
	// ModuleName is the name of the settlement module.
	ModuleName = "settlement"

	// StoreKey is the store key string for settlement.
	StoreKey = ModuleName

	// RouterKey is the message route for settlement.
	RouterKey = ModuleName

	// QuerierRoute is the querier route for settlement.
	QuerierRoute = ModuleName
)

// Key prefixes for the settlement store.
var (
	EscrowKeyPrefix       = []byte{0x01}
	EscrowByCreatorPrefix = []byte{0x02}
	BondingCurvePrefix    = []byte{0x03}
	ParamsKey             = []byte{0x04}
	EscrowCounterKey      = []byte{0x05}
)

// EscrowKey returns the store key for a specific escrow by ID.
func EscrowKey(escrowID string) []byte {
	return append(EscrowKeyPrefix, []byte(escrowID)...)
}

// EscrowByCreatorKey returns the store key for escrows by creator.
func EscrowByCreatorKey(creator, escrowID string) []byte {
	key := append(EscrowByCreatorPrefix, []byte(creator)...)
	key = append(key, '/')
	key = append(key, []byte(escrowID)...)
	return key
}

// EscrowByCreatorIteratorPrefix returns the prefix for iterating escrows by a creator.
func EscrowByCreatorIteratorPrefix(creator string) []byte {
	key := append(EscrowByCreatorPrefix, []byte(creator)...)
	key = append(key, '/')
	return key
}

// BondingCurveKey returns the store key for a bonding curve state by asset ID.
func BondingCurveKey(assetID string) []byte {
	return append(BondingCurvePrefix, []byte(assetID)...)
}
