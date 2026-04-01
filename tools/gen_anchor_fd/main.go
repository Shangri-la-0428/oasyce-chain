// Generates proper gzipped FileDescriptorProto bytes for the anchor module's pb.go files.
// Usage: go run ./tools/gen_anchor_fd
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	base := "x/anchor/types"

	patches := []struct {
		file    string
		varName string
		fd      *descriptorpb.FileDescriptorProto
	}{
		{
			file:    "types.pb.go",
			varName: "fileDescriptor_anchor_types",
			fd:      typesDescriptor(),
		},
		{
			file:    "genesis.pb.go",
			varName: "fileDescriptor_anchor_genesis",
			fd:      genesisDescriptor(),
		},
		{
			file:    "tx.pb.go",
			varName: "fileDescriptor_anchor_tx",
			fd:      txDescriptor(),
		},
		{
			file:    "query.pb.go",
			varName: "fileDescriptor_anchor_query",
			fd:      queryDescriptor(),
		},
	}

	for _, p := range patches {
		gz := gzipFD(p.fd)
		path := filepath.Join(base, p.file)
		patchFile(path, p.varName, gz)
		fmt.Printf("Patched %s (%s): %d bytes\n", path, p.varName, len(gz))
	}
}

func typesDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("oasyce/anchor/v1/types.proto"),
		Package: proto.String("oasyce.anchor.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("AnchorRecord"),
				Field: []*descriptorpb.FieldDescriptorProto{
					bytesField("trace_id", 1),
					bytesField("node_pubkey", 2),
					stringField("capability", 3),
					uint32Field("outcome", 4),
					uint64Field("timestamp", 5),
					int64Field("anchor_height", 6),
					bytesField("trace_signature", 7),
				},
			},
		},
	}
}

func genesisDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String("oasyce/anchor/v1/genesis.proto"),
		Package:    proto.String("oasyce.anchor.v1"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"oasyce/anchor/v1/types.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("GenesisState"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("anchors"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".oasyce.anchor.v1.AnchorRecord"),
						JsonName: proto.String("anchors"),
					},
				},
			},
		},
	}
}

func txDescriptor() *descriptorpb.FileDescriptorProto {
	// Build message options with proper extension field encoding
	// (not UninterpretedOption — Cosmos SDK reads raw extension field 11110000)
	anchorTraceOpts := &descriptorpb.MessageOptions{}
	anchorTraceOpts.ProtoReflect().SetUnknown(buildSignerOptions("signer", "oasyce/anchor/MsgAnchorTrace"))

	anchorBatchOpts := &descriptorpb.MessageOptions{}
	anchorBatchOpts.ProtoReflect().SetUnknown(buildSignerOptions("signer", "oasyce/anchor/MsgAnchorBatch"))

	// Service-level cosmos.msg.v1.service option (field 11110000, empty value)
	svcOpts := &descriptorpb.ServiceOptions{}
	svcOpts.ProtoReflect().SetUnknown(buildServiceOption())

	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("oasyce/anchor/v1/tx.proto"),
		Package: proto.String("oasyce.anchor.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("MsgAnchorTrace"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("signer", 1),
					bytesField("trace_id", 2),
					bytesField("node_pubkey", 3),
					stringField("capability", 4),
					uint32Field("outcome", 5),
					uint64Field("timestamp", 6),
					bytesField("trace_signature", 7),
				},
				Options: anchorTraceOpts,
			},
			emptyMessage("MsgAnchorTraceResponse"),
			{
				Name: proto.String("MsgAnchorBatch"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("signer", 1),
					{
						Name:     proto.String("anchors"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".oasyce.anchor.v1.MsgAnchorTrace"),
						JsonName: proto.String("anchors"),
					},
				},
				Options: anchorBatchOpts,
			},
			{
				Name: proto.String("MsgAnchorBatchResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					uint32Field("anchored", 1),
					uint32Field("skipped", 2),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Msg"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("AnchorTrace"),
						InputType:  proto.String(".oasyce.anchor.v1.MsgAnchorTrace"),
						OutputType: proto.String(".oasyce.anchor.v1.MsgAnchorTraceResponse"),
					},
					{
						Name:       proto.String("AnchorBatch"),
						InputType:  proto.String(".oasyce.anchor.v1.MsgAnchorBatch"),
						OutputType: proto.String(".oasyce.anchor.v1.MsgAnchorBatchResponse"),
					},
				},
				Options: svcOpts,
			},
		},
	}
}

// buildSignerOptions constructs raw protobuf bytes for cosmos.msg.v1.signer and amino options.
// Extension field 11110000 = cosmos.msg.v1.signer (string, wire type 2)
// Extension field 11110001 = amino type URL (string, wire type 2)
func buildSignerOptions(signer, aminoURL string) []byte {
	var buf []byte
	// Field 11110000, wire type 2 (LDel)
	buf = appendVarint(buf, (11110000<<3)|2)
	buf = appendVarint(buf, uint64(len(signer)))
	buf = append(buf, signer...)
	// Field 11110001, wire type 2 (LDel)
	buf = appendVarint(buf, (11110001<<3)|2)
	buf = appendVarint(buf, uint64(len(aminoURL)))
	buf = append(buf, aminoURL...)
	return buf
}

// buildServiceOption constructs raw protobuf bytes for cosmos.msg.v1.service option.
// This is a bool-like marker — field 11110000, varint value 0 (or just the tag).
func buildServiceOption() []byte {
	var buf []byte
	// Field 11110000, wire type 0 (varint), value = 0 (present = true)
	buf = appendVarint(buf, (11110000<<3)|0)
	buf = append(buf, 0)
	return buf
}

func appendVarint(buf []byte, x uint64) []byte {
	for x >= 0x80 {
		buf = append(buf, byte(x)|0x80)
		x >>= 7
	}
	return append(buf, byte(x))
}

func queryDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String("oasyce/anchor/v1/query.proto"),
		Package:    proto.String("oasyce.anchor.v1"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"oasyce/anchor/v1/types.proto", "cosmos/base/query/v1beta1/pagination.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("QueryAnchorRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					bytesField("trace_id", 1),
				},
			},
			{
				Name: proto.String("QueryAnchorResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("anchor"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".oasyce.anchor.v1.AnchorRecord"),
						JsonName: proto.String("anchor"),
					},
				},
			},
			{
				Name: proto.String("QueryIsAnchoredRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					bytesField("trace_id", 1),
				},
			},
			{
				Name: proto.String("QueryIsAnchoredResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					boolField("anchored", 1),
				},
			},
			{
				Name: proto.String("QueryAnchorsByCapabilityRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("capability", 1),
					{
						Name:     proto.String("pagination"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".cosmos.base.query.v1beta1.PageRequest"),
						JsonName: proto.String("pagination"),
					},
				},
			},
			{
				Name: proto.String("QueryAnchorsByCapabilityResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("anchors"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".oasyce.anchor.v1.AnchorRecord"),
						JsonName: proto.String("anchors"),
					},
					{
						Name:     proto.String("pagination"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".cosmos.base.query.v1beta1.PageResponse"),
						JsonName: proto.String("pagination"),
					},
				},
			},
			{
				Name: proto.String("QueryAnchorsByNodeRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					bytesField("node_pubkey", 1),
					{
						Name:     proto.String("pagination"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".cosmos.base.query.v1beta1.PageRequest"),
						JsonName: proto.String("pagination"),
					},
				},
			},
			{
				Name: proto.String("QueryAnchorsByNodeResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("anchors"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".oasyce.anchor.v1.AnchorRecord"),
						JsonName: proto.String("anchors"),
					},
					{
						Name:     proto.String("pagination"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".cosmos.base.query.v1beta1.PageResponse"),
						JsonName: proto.String("pagination"),
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Query"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("Anchor"),
						InputType:  proto.String(".oasyce.anchor.v1.QueryAnchorRequest"),
						OutputType: proto.String(".oasyce.anchor.v1.QueryAnchorResponse"),
					},
					{
						Name:       proto.String("IsAnchored"),
						InputType:  proto.String(".oasyce.anchor.v1.QueryIsAnchoredRequest"),
						OutputType: proto.String(".oasyce.anchor.v1.QueryIsAnchoredResponse"),
					},
					{
						Name:       proto.String("AnchorsByCapability"),
						InputType:  proto.String(".oasyce.anchor.v1.QueryAnchorsByCapabilityRequest"),
						OutputType: proto.String(".oasyce.anchor.v1.QueryAnchorsByCapabilityResponse"),
					},
					{
						Name:       proto.String("AnchorsByNode"),
						InputType:  proto.String(".oasyce.anchor.v1.QueryAnchorsByNodeRequest"),
						OutputType: proto.String(".oasyce.anchor.v1.QueryAnchorsByNodeResponse"),
					},
				},
			},
		},
	}
}

// --- field helpers ---

func stringField(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		JsonName: proto.String(name),
	}
}

func bytesField(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
		JsonName: proto.String(name),
	}
}

func uint32Field(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(),
		JsonName: proto.String(name),
	}
}

func uint64Field(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
		JsonName: proto.String(name),
	}
}

func int64Field(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
		JsonName: proto.String(name),
	}
}

func boolField(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
		JsonName: proto.String(name),
	}
}

func emptyMessage(name string) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{Name: proto.String(name)}
}

// --- gzip + patch helpers ---

func gzipFD(fd *descriptorpb.FileDescriptorProto) []byte {
	b, err := proto.Marshal(fd)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		panic(err)
	}
	w.Close()
	return buf.Bytes()
}

func patchFile(path string, varName string, newBytes []byte) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("cannot read %s: %v", path, err))
	}

	// Match: var varName = []byte{ ... }
	pattern := regexp.MustCompile(
		`(?s)(var\s+` + regexp.QuoteMeta(varName) + `\s*=\s*\[\]byte\{)[^}]*(\})`,
	)

	replacement := "$1\n" + formatBytes(newBytes) + "\n$2"
	result := pattern.ReplaceAll(data, []byte(replacement))

	if bytes.Equal(data, result) {
		fmt.Printf("WARNING: no match for %s in %s\n", varName, path)
		return
	}

	if err := os.WriteFile(path, result, 0644); err != nil {
		panic(fmt.Sprintf("cannot write %s: %v", path, err))
	}
}

func formatBytes(b []byte) string {
	var lines []string
	for i := 0; i < len(b); i += 16 {
		end := i + 16
		if end > len(b) {
			end = len(b)
		}
		var hexes []string
		for _, v := range b[i:end] {
			hexes = append(hexes, fmt.Sprintf("0x%02x", v))
		}
		lines = append(lines, "\t"+strings.Join(hexes, ", ")+",")
	}
	return strings.Join(lines, "\n")
}
