package types

import "encoding/binary"

const (
	ModuleName = "sigil"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

// Store key prefixes.
var (
	SigilKeyPrefix        = []byte{0x01} // sigil_id -> Sigil
	BondKeyPrefix         = []byte{0x02} // bond_id -> Bond
	BondsBySigilPrefix    = []byte{0x03} // sigil_id/bond_id -> bond_id (index)
	LineagePrefix         = []byte{0x04} // parent_sigil_id/child_sigil_id -> child_sigil_id
	LivenessIndexPrefix   = []byte{0x05} // last_active_height(8) + sigil_id -> sigil_id
	ParamsKey             = []byte{0x06}
	ActiveCountKey        = []byte{0x07}
	SigilByStatusPrefix   = []byte{0x08} // status(1) + sigil_id -> sigil_id (index)
)

func SigilKey(sigilID string) []byte {
	return append(SigilKeyPrefix, []byte(sigilID)...)
}

func BondKey(bondID string) []byte {
	return append(BondKeyPrefix, []byte(bondID)...)
}

func BondsBySigilKey(sigilID, bondID string) []byte {
	key := append(BondsBySigilPrefix, []byte(sigilID)...)
	key = append(key, '/')
	key = append(key, []byte(bondID)...)
	return key
}

func BondsBySigilIteratorPrefix(sigilID string) []byte {
	key := append(BondsBySigilPrefix, []byte(sigilID)...)
	key = append(key, '/')
	return key
}

func LineageKey(parentID, childID string) []byte {
	key := append(LineagePrefix, []byte(parentID)...)
	key = append(key, '/')
	key = append(key, []byte(childID)...)
	return key
}

func LineageIteratorPrefix(parentID string) []byte {
	key := append(LineagePrefix, []byte(parentID)...)
	key = append(key, '/')
	return key
}

func LivenessIndexKey(height int64, sigilID string) []byte {
	key := make([]byte, len(LivenessIndexPrefix)+8+len(sigilID))
	copy(key, LivenessIndexPrefix)
	binary.BigEndian.PutUint64(key[len(LivenessIndexPrefix):], uint64(height))
	copy(key[len(LivenessIndexPrefix)+8:], sigilID)
	return key
}

func LivenessIndexIteratorPrefix(maxHeight int64) []byte {
	// Returns prefix for scanning all entries with height <= maxHeight.
	// We scan from LivenessIndexPrefix to LivenessIndexPrefix + (maxHeight+1).
	key := make([]byte, len(LivenessIndexPrefix)+8)
	copy(key, LivenessIndexPrefix)
	binary.BigEndian.PutUint64(key[len(LivenessIndexPrefix):], uint64(maxHeight+1))
	return key
}

func SigilByStatusKey(status SigilStatus, sigilID string) []byte {
	key := append(SigilByStatusPrefix, byte(status))
	key = append(key, []byte(sigilID)...)
	return key
}

func SigilByStatusIteratorPrefix(status SigilStatus) []byte {
	return append(SigilByStatusPrefix, byte(status))
}
