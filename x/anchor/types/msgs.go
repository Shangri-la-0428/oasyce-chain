package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Ensure proto-generated message types implement sdk.Msg.
var (
	_ sdk.Msg = &MsgAnchorTrace{}
	_ sdk.Msg = &MsgAnchorBatch{}
)

// MaxBatchSize is the maximum number of anchors in a single batch.
const MaxBatchSize = 50

// ValidateBasic performs stateless validation for MsgAnchorTrace.
func (msg *MsgAnchorTrace) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrInvalidAddress.Wrapf("invalid signer address: %s", err)
	}
	if len(msg.TraceId) == 0 {
		return ErrInvalidTraceID.Wrap("trace_id cannot be empty")
	}
	if len(msg.TraceId) > 64 {
		return ErrInvalidTraceID.Wrap("trace_id too long (max 64 bytes)")
	}
	if len(msg.NodePubkey) != 32 {
		return ErrInvalidPubkey.Wrapf("node_pubkey must be 32 bytes (ed25519), got %d", len(msg.NodePubkey))
	}
	if msg.Capability == "" {
		return ErrInvalidCapability.Wrap("capability cannot be empty")
	}
	if len(msg.Capability) > 256 {
		return ErrInvalidCapability.Wrap("capability too long (max 256 chars)")
	}
	if msg.Timestamp == 0 {
		return ErrInvalidTimestamp.Wrap("timestamp cannot be zero")
	}
	if len(msg.TraceSignature) != 64 {
		return ErrInvalidSignature.Wrapf("trace_signature must be 64 bytes (ed25519), got %d", len(msg.TraceSignature))
	}
	return nil
}

// ValidateBasic performs stateless validation for MsgAnchorBatch.
func (msg *MsgAnchorBatch) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrInvalidAddress.Wrapf("invalid signer address: %s", err)
	}
	if len(msg.Anchors) == 0 {
		return ErrInvalidTraceID.Wrap("batch cannot be empty")
	}
	if len(msg.Anchors) > MaxBatchSize {
		return ErrBatchTooLarge.Wrapf("batch has %d anchors, max %d", len(msg.Anchors), MaxBatchSize)
	}
	for i, anchor := range msg.Anchors {
		if err := anchor.ValidateBasic(); err != nil {
			return ErrInvalidTraceID.Wrapf("anchor[%d]: %s", i, err)
		}
	}
	return nil
}
