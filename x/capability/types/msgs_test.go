package types_test

import (
	"strings"
	"testing"

	"github.com/oasyce/chain/x/capability/types"
)

const validAddr = "cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"

// ---------- MsgCompleteInvocation ----------

func TestMsgCompleteInvocationValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgCompleteInvocation
		wantErr bool
	}{
		{
			name: "valid message",
			msg: types.MsgCompleteInvocation{
				Creator:      validAddr,
				InvocationId: "INV_001",
				OutputHash:   strings.Repeat("a", 32),
			},
			wantErr: false,
		},
		{
			name: "empty creator",
			msg: types.MsgCompleteInvocation{
				Creator:      "",
				InvocationId: "INV_001",
				OutputHash:   strings.Repeat("a", 32),
			},
			wantErr: true,
		},
		{
			name: "invalid bech32 creator",
			msg: types.MsgCompleteInvocation{
				Creator:      "notabech32address",
				InvocationId: "INV_001",
				OutputHash:   strings.Repeat("a", 32),
			},
			wantErr: true,
		},
		{
			name: "empty invocation_id",
			msg: types.MsgCompleteInvocation{
				Creator:      validAddr,
				InvocationId: "",
				OutputHash:   strings.Repeat("a", 32),
			},
			wantErr: true,
		},
		{
			name: "output hash too short (31 chars)",
			msg: types.MsgCompleteInvocation{
				Creator:      validAddr,
				InvocationId: "INV_001",
				OutputHash:   strings.Repeat("a", 31),
			},
			wantErr: true,
		},
		{
			name: "output hash exactly 32 chars",
			msg: types.MsgCompleteInvocation{
				Creator:      validAddr,
				InvocationId: "INV_001",
				OutputHash:   strings.Repeat("b", 32),
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------- MsgClaimInvocation ----------

func TestMsgClaimInvocationValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgClaimInvocation
		wantErr bool
	}{
		{
			name: "valid message",
			msg: types.MsgClaimInvocation{
				Creator:      validAddr,
				InvocationId: "INV_001",
			},
			wantErr: false,
		},
		{
			name: "empty creator",
			msg: types.MsgClaimInvocation{
				Creator:      "",
				InvocationId: "INV_001",
			},
			wantErr: true,
		},
		{
			name: "empty invocation_id",
			msg: types.MsgClaimInvocation{
				Creator:      validAddr,
				InvocationId: "",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------- MsgDisputeInvocation ----------

func TestMsgDisputeInvocationValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgDisputeInvocation
		wantErr bool
	}{
		{
			name: "valid message",
			msg: types.MsgDisputeInvocation{
				Creator:      validAddr,
				InvocationId: "INV_001",
				Reason:       "output was incorrect",
			},
			wantErr: false,
		},
		{
			name: "empty creator",
			msg: types.MsgDisputeInvocation{
				Creator:      "",
				InvocationId: "INV_001",
				Reason:       "output was incorrect",
			},
			wantErr: true,
		},
		{
			name: "empty invocation_id",
			msg: types.MsgDisputeInvocation{
				Creator:      validAddr,
				InvocationId: "",
				Reason:       "output was incorrect",
			},
			wantErr: true,
		},
		{
			name: "empty reason",
			msg: types.MsgDisputeInvocation{
				Creator:      validAddr,
				InvocationId: "INV_001",
				Reason:       "",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------- MsgFailInvocation ----------

func TestMsgFailInvocationValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgFailInvocation
		wantErr bool
	}{
		{
			name: "valid message",
			msg: types.MsgFailInvocation{
				Creator:      validAddr,
				InvocationId: "INV_001",
			},
			wantErr: false,
		},
		{
			name: "empty creator",
			msg: types.MsgFailInvocation{
				Creator:      "",
				InvocationId: "INV_001",
			},
			wantErr: true,
		},
		{
			name: "empty invocation_id",
			msg: types.MsgFailInvocation{
				Creator:      validAddr,
				InvocationId: "",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
