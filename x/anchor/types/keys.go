package types

const (
	// ModuleName is the name of the anchor module.
	ModuleName = "anchor"

	// StoreKey is the store key string for anchor.
	StoreKey = ModuleName

	// RouterKey is the message route for anchor.
	RouterKey = ModuleName

	// QuerierRoute is the querier route for anchor.
	QuerierRoute = ModuleName
)

// Key prefixes for the anchor store.
var (
	AnchorKeyPrefix     = []byte{0x01} // anchor/ -> AnchorRecord by trace_id
	AnchorByCapPrefix   = []byte{0x02} // anchor_by_cap/ -> trace_id index by capability
	AnchorByNodePrefix  = []byte{0x03} // anchor_by_node/ -> trace_id index by node pubkey
)

// AnchorKey returns the store key for an anchor record by trace_id.
func AnchorKey(traceID []byte) []byte {
	return append(AnchorKeyPrefix, traceID...)
}

// AnchorByCapKey returns the store key for indexing an anchor by capability.
func AnchorByCapKey(capability string, traceID []byte) []byte {
	key := append(AnchorByCapPrefix, []byte(capability)...)
	key = append(key, '/')
	key = append(key, traceID...)
	return key
}

// AnchorByCapIteratorPrefix returns the prefix for iterating anchors by capability.
func AnchorByCapIteratorPrefix(capability string) []byte {
	key := append(AnchorByCapPrefix, []byte(capability)...)
	key = append(key, '/')
	return key
}

// AnchorByNodeKey returns the store key for indexing an anchor by node pubkey.
func AnchorByNodeKey(nodePubkey []byte, traceID []byte) []byte {
	key := append(AnchorByNodePrefix, nodePubkey...)
	key = append(key, '/')
	key = append(key, traceID...)
	return key
}

// AnchorByNodeIteratorPrefix returns the prefix for iterating anchors by node pubkey.
func AnchorByNodeIteratorPrefix(nodePubkey []byte) []byte {
	key := append(AnchorByNodePrefix, nodePubkey...)
	key = append(key, '/')
	return key
}
