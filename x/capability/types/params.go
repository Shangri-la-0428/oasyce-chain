package types

import (
	proto "github.com/cosmos/gogoproto/proto"
)

// String implements the proto.Message interface for Params.
// This method was missing from the generated genesis.pb.go.
func (m *Params) String() string { return proto.CompactTextString(m) }
