package types

const (
	// ModuleName is the name of the datarights module.
	ModuleName = "datarights"

	// StoreKey is the store key string for datarights.
	StoreKey = ModuleName

	// RouterKey is the message route for datarights.
	RouterKey = ModuleName

	// QuerierRoute is the querier route for datarights.
	QuerierRoute = ModuleName
)

// Key prefixes for the datarights store.
var (
	DataAssetKeyPrefix       = []byte{0x01}
	ShareHolderKeyPrefix     = []byte{0x02}
	DisputeKeyPrefix         = []byte{0x03}
	ParamsKey                = []byte{0x04}
	AssetCounterKey          = []byte{0x05}
	DisputeCounterKey        = []byte{0x06}
	AssetByOwnerPrefix       = []byte{0x07}
	ShareHolderByAssetPrefix = []byte{0x08}
	AssetReservePrefix       = []byte{0x09}
)

// AssetReserveKey returns the store key for the bonding curve reserve of an asset.
func AssetReserveKey(assetID string) []byte {
	return append(AssetReservePrefix, []byte(assetID)...)
}

// DataAssetKey returns the store key for a specific data asset by ID.
func DataAssetKey(assetID string) []byte {
	return append(DataAssetKeyPrefix, []byte(assetID)...)
}

// AssetByOwnerKey returns the store key for assets by owner.
func AssetByOwnerKey(owner, assetID string) []byte {
	key := append(AssetByOwnerPrefix, []byte(owner)...)
	key = append(key, '/')
	key = append(key, []byte(assetID)...)
	return key
}

// AssetByOwnerIteratorPrefix returns the prefix for iterating assets by owner.
func AssetByOwnerIteratorPrefix(owner string) []byte {
	key := append(AssetByOwnerPrefix, []byte(owner)...)
	key = append(key, '/')
	return key
}

// ShareHolderKey returns the store key for a shareholder record.
func ShareHolderKey(assetID, address string) []byte {
	key := append(ShareHolderKeyPrefix, []byte(assetID)...)
	key = append(key, '/')
	key = append(key, []byte(address)...)
	return key
}

// ShareHolderByAssetIteratorPrefix returns the prefix for iterating shareholders by asset.
func ShareHolderByAssetIteratorPrefix(assetID string) []byte {
	key := append(ShareHolderByAssetPrefix, []byte(assetID)...)
	key = append(key, '/')
	return key
}

// ShareHolderByAssetKey returns the secondary index key for shareholders by asset.
func ShareHolderByAssetKey(assetID, address string) []byte {
	key := append(ShareHolderByAssetPrefix, []byte(assetID)...)
	key = append(key, '/')
	key = append(key, []byte(address)...)
	return key
}

// DisputeKey returns the store key for a specific dispute by ID.
func DisputeKey(disputeID string) []byte {
	return append(DisputeKeyPrefix, []byte(disputeID)...)
}
