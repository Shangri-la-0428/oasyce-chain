// Generates gzipped proto file descriptor bytes for x/sigil tx.proto and query.proto.
// Usage: go run ./tools/gen_sigil_descriptors
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	typesFd := buildTypesDescriptor()
	fmt.Println("// --- fileDescriptor_sigil_types ---")
	printGoBytes("fileDescriptor_sigil_types", gzipDescriptor(typesFd))

	fmt.Println()

	txFd := buildTxDescriptor()
	fmt.Println("// --- fileDescriptor_sigil_tx ---")
	printGoBytes("fileDescriptor_sigil_tx", gzipDescriptor(txFd))

	fmt.Println()

	queryFd := buildQueryDescriptor()
	fmt.Println("// --- fileDescriptor_sigil_query ---")
	printGoBytes("fileDescriptor_sigil_query", gzipDescriptor(queryFd))
}

// signerOpts returns MessageOptions with cosmos.msg.v1.signer extension (field 11110000).
func signerOpts(fieldName string) *descriptorpb.MessageOptions {
	opts := &descriptorpb.MessageOptions{}
	// Encode extension: field 11110000, wire type 2 (bytes), value = fieldName
	var raw []byte
	raw = protowire.AppendTag(raw, 11110000, protowire.BytesType)
	raw = protowire.AppendBytes(raw, []byte(fieldName))
	opts.ProtoReflect().SetUnknown(raw)
	return opts
}

func stringField(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		JsonName: proto.String(jsonName(name)),
	}
}

func bytesField(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
		JsonName: proto.String(jsonName(name)),
	}
}

func int32Field(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
		JsonName: proto.String(jsonName(name)),
	}
}

func int64Field(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
		JsonName: proto.String(jsonName(name)),
	}
}

func repeatedStringField(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		JsonName: proto.String(jsonName(name)),
	}
}

func msgField(name string, num int32, typeName string) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		TypeName: proto.String(typeName),
		JsonName: proto.String(jsonName(name)),
	}
}

func boolField(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
		JsonName: proto.String(jsonName(name)),
	}
}

func uint64Field(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
		JsonName: proto.String(jsonName(name)),
	}
}

func repeatedMsgField(name string, num int32, typeName string) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		TypeName: proto.String(typeName),
		JsonName: proto.String(jsonName(name)),
	}
}

func emptyMsg(name string) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{Name: proto.String(name)}
}

func rpc(name, input, output string) *descriptorpb.MethodDescriptorProto {
	return &descriptorpb.MethodDescriptorProto{
		Name:       proto.String(name),
		InputType:  proto.String(input),
		OutputType: proto.String(output),
	}
}

func jsonName(s string) string {
	// Simple snake_case to camelCase
	out := make([]byte, 0, len(s))
	upper := false
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			upper = true
			continue
		}
		if upper {
			if s[i] >= 'a' && s[i] <= 'z' {
				out = append(out, s[i]-32)
			} else {
				out = append(out, s[i])
			}
			upper = false
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}

func buildTypesDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("oasyce/sigil/v1/types.proto"),
		Package: proto.String("oasyce.sigil.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			// 0: Sigil
			{
				Name: proto.String("Sigil"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("sigil_id", 1),
					stringField("creator", 2),
					bytesField("public_key", 3),
					int32Field("status", 4),
					int64Field("creation_height", 5),
					int64Field("last_active_height", 6),
					bytesField("state_root", 7),
					repeatedStringField("lineage", 8),
					stringField("metadata", 9),
				},
			},
			// 1: Bond
			{
				Name: proto.String("Bond"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("bond_id", 1),
					stringField("sigil_a", 2),
					stringField("sigil_b", 3),
					bytesField("terms_hash", 4),
					int64Field("creation_height", 5),
					stringField("scope", 6),
				},
			},
			// 2: Params
			{
				Name: proto.String("Params"),
				Field: []*descriptorpb.FieldDescriptorProto{
					int64Field("dormant_threshold", 1),
					int64Field("dissolve_threshold", 2),
					int64Field("submit_window", 3),
				},
			},
			// 3: GenesisState
			{
				Name: proto.String("GenesisState"),
				Field: []*descriptorpb.FieldDescriptorProto{
					repeatedMsgField("sigils", 1, ".oasyce.sigil.v1.Sigil"),
					repeatedMsgField("bonds", 2, ".oasyce.sigil.v1.Bond"),
					msgField("params", 3, ".oasyce.sigil.v1.Params"),
				},
			},
		},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/oasyce/chain/x/sigil/types"),
		},
	}
}

func buildTxDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("oasyce/sigil/v1/tx.proto"),
		Package: proto.String("oasyce.sigil.v1"),
		Syntax:  proto.String("proto3"),
		Dependency: []string{
			"cosmos/msg/v1/msg.proto",
		},
		MessageType: []*descriptorpb.DescriptorProto{
			// 0: MsgGenesis
			{
				Name:    proto.String("MsgGenesis"),
				Options: signerOpts("signer"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("signer", 1),
					bytesField("public_key", 2),
					repeatedStringField("lineage", 3),
					bytesField("state_root", 4),
					stringField("metadata", 5),
				},
			},
			// 1: MsgGenesisResponse
			{
				Name: proto.String("MsgGenesisResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("sigil_id", 1),
				},
			},
			// 2: MsgDissolve
			{
				Name:    proto.String("MsgDissolve"),
				Options: signerOpts("signer"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("signer", 1),
					stringField("sigil_id", 2),
				},
			},
			// 3: MsgDissolveResponse
			emptyMsg("MsgDissolveResponse"),
			// 4: MsgBond
			{
				Name:    proto.String("MsgBond"),
				Options: signerOpts("signer"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("signer", 1),
					stringField("sigil_a", 2),
					stringField("sigil_b", 3),
					bytesField("terms_hash", 4),
					stringField("scope", 5),
				},
			},
			// 5: MsgBondResponse
			{
				Name: proto.String("MsgBondResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("bond_id", 1),
				},
			},
			// 6: MsgUnbond
			{
				Name:    proto.String("MsgUnbond"),
				Options: signerOpts("signer"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("signer", 1),
					stringField("bond_id", 2),
				},
			},
			// 7: MsgUnbondResponse
			emptyMsg("MsgUnbondResponse"),
			// 8: MsgFork
			{
				Name:    proto.String("MsgFork"),
				Options: signerOpts("signer"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("signer", 1),
					stringField("parent_sigil_id", 2),
					bytesField("public_key", 3),
					int32Field("fork_mode", 4),
					stringField("mutation", 5),
					stringField("metadata", 6),
				},
			},
			// 9: MsgForkResponse
			{
				Name: proto.String("MsgForkResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("child_sigil_id", 1),
				},
			},
			// 10: MsgMerge
			{
				Name:    proto.String("MsgMerge"),
				Options: signerOpts("signer"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("signer", 1),
					stringField("sigil_a", 2),
					stringField("sigil_b", 3),
					int32Field("merge_mode", 4),
					stringField("metadata", 5),
				},
			},
			// 11: MsgMergeResponse
			{
				Name: proto.String("MsgMergeResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("merged_sigil_id", 1),
				},
			},
			// 12: MsgUpdateParams
			{
				Name:    proto.String("MsgUpdateParams"),
				Options: signerOpts("authority"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("authority", 1),
					msgField("params", 2, ".oasyce.sigil.v1.Params"),
				},
			},
			// 13: MsgUpdateParamsResponse
			emptyMsg("MsgUpdateParamsResponse"),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Msg"),
				Method: []*descriptorpb.MethodDescriptorProto{
					rpc("Genesis", ".oasyce.sigil.v1.MsgGenesis", ".oasyce.sigil.v1.MsgGenesisResponse"),
					rpc("Dissolve", ".oasyce.sigil.v1.MsgDissolve", ".oasyce.sigil.v1.MsgDissolveResponse"),
					rpc("Bond", ".oasyce.sigil.v1.MsgBond", ".oasyce.sigil.v1.MsgBondResponse"),
					rpc("Unbond", ".oasyce.sigil.v1.MsgUnbond", ".oasyce.sigil.v1.MsgUnbondResponse"),
					rpc("Fork", ".oasyce.sigil.v1.MsgFork", ".oasyce.sigil.v1.MsgForkResponse"),
					rpc("Merge", ".oasyce.sigil.v1.MsgMerge", ".oasyce.sigil.v1.MsgMergeResponse"),
					rpc("UpdateParams", ".oasyce.sigil.v1.MsgUpdateParams", ".oasyce.sigil.v1.MsgUpdateParamsResponse"),
				},
			},
		},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/oasyce/chain/x/sigil/types"),
		},
	}
}

func buildQueryDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("oasyce/sigil/v1/query.proto"),
		Package: proto.String("oasyce.sigil.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			// 0: QuerySigilRequest
			{
				Name: proto.String("QuerySigilRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("sigil_id", 1),
				},
			},
			// 1: QuerySigilResponse
			{
				Name: proto.String("QuerySigilResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					msgField("sigil", 1, ".oasyce.sigil.v1.Sigil"),
				},
			},
			// 2: QueryBondRequest
			{
				Name: proto.String("QueryBondRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("bond_id", 1),
				},
			},
			// 3: QueryBondResponse
			{
				Name: proto.String("QueryBondResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					msgField("bond", 1, ".oasyce.sigil.v1.Bond"),
				},
			},
			// 4: QueryBondsBySigilRequest
			{
				Name: proto.String("QueryBondsBySigilRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("sigil_id", 1),
				},
			},
			// 5: QueryBondsBySigilResponse
			{
				Name: proto.String("QueryBondsBySigilResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					repeatedMsgField("bonds", 1, ".oasyce.sigil.v1.Bond"),
				},
			},
			// 6: QueryLineageRequest
			{
				Name: proto.String("QueryLineageRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					stringField("sigil_id", 1),
				},
			},
			// 7: QueryLineageResponse
			{
				Name: proto.String("QueryLineageResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					repeatedStringField("children", 1),
				},
			},
			// 8: QueryActiveCountRequest
			emptyMsg("QueryActiveCountRequest"),
			// 9: QueryActiveCountResponse
			{
				Name: proto.String("QueryActiveCountResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					uint64Field("count", 1),
				},
			},
			// 10: QueryParamsRequest
			emptyMsg("QueryParamsRequest"),
			// 11: QueryParamsResponse
			{
				Name: proto.String("QueryParamsResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					msgField("params", 1, ".oasyce.sigil.v1.Params"),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Query"),
				Method: []*descriptorpb.MethodDescriptorProto{
					rpc("Sigil", ".oasyce.sigil.v1.QuerySigilRequest", ".oasyce.sigil.v1.QuerySigilResponse"),
					rpc("Bond", ".oasyce.sigil.v1.QueryBondRequest", ".oasyce.sigil.v1.QueryBondResponse"),
					rpc("BondsBySigil", ".oasyce.sigil.v1.QueryBondsBySigilRequest", ".oasyce.sigil.v1.QueryBondsBySigilResponse"),
					rpc("Lineage", ".oasyce.sigil.v1.QueryLineageRequest", ".oasyce.sigil.v1.QueryLineageResponse"),
					rpc("ActiveCount", ".oasyce.sigil.v1.QueryActiveCountRequest", ".oasyce.sigil.v1.QueryActiveCountResponse"),
					rpc("Params", ".oasyce.sigil.v1.QueryParamsRequest", ".oasyce.sigil.v1.QueryParamsResponse"),
				},
			},
		},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/oasyce/chain/x/sigil/types"),
		},
	}
}

func gzipDescriptor(fd *descriptorpb.FileDescriptorProto) []byte {
	b, err := proto.Marshal(fd)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func printGoBytes(name string, data []byte) {
	fmt.Printf("var %s = []byte{\n\t", name)
	for i, b := range data {
		if i > 0 && i%16 == 0 {
			fmt.Print("\n\t")
		}
		fmt.Printf("0x%02x, ", b)
	}
	fmt.Println("\n}")
}
