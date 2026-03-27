package types

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Anchors: []AnchorRecord{},
	}
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(gs GenesisState) error {
	seen := make(map[string]bool)
	for _, anchor := range gs.Anchors {
		key := string(anchor.TraceId)
		if seen[key] {
			return ErrDuplicateAnchor.Wrapf("duplicate trace_id in genesis: %x", anchor.TraceId)
		}
		seen[key] = true

		if len(anchor.TraceId) == 0 {
			return ErrInvalidTraceID.Wrap("trace_id cannot be empty in genesis")
		}
		if len(anchor.NodePubkey) != 32 {
			return ErrInvalidPubkey.Wrapf("node_pubkey must be 32 bytes, got %d", len(anchor.NodePubkey))
		}
		if anchor.Capability == "" {
			return ErrInvalidCapability.Wrap("capability cannot be empty in genesis")
		}
	}
	return nil
}
