// Tool to patch proto file descriptors in tx.pb.go files.
// Adds missing RPC methods and message types to each module's file descriptor.
//
// For each module, the tool reads the Go _Msg_serviceDesc to determine which
// methods exist in code, then patches the file descriptor to match.
//
// Usage: go run ./tools/patch_descriptors
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

type moduleInfo struct {
	dir       string
	pbFile    string
	fdVarName string
}

var modules = []moduleInfo{
	{dir: "x/settlement/types", pbFile: "tx.pb.go", fdVarName: "fileDescriptor_8cbf2bca1e5934e1"},
	{dir: "x/capability/types", pbFile: "tx.pb.go", fdVarName: "fileDescriptor_d5b895fbb0c07113"},
	{dir: "x/reputation/types", pbFile: "tx.pb.go", fdVarName: "fileDescriptor_a6fb122e1b6f616e"},
	{dir: "x/datarights/types", pbFile: "tx.pb.go", fdVarName: "fileDescriptor_f4554f457b53c7be"},
	{dir: "x/work/types", pbFile: "tx.pb.go", fdVarName: "fileDescriptor_6b9943cdc07cfdd8"},
	{dir: "x/onboarding/types", pbFile: "tx.pb.go", fdVarName: "fileDescriptor_c794ac0b330f1318"},
	{dir: "x/anchor/types", pbFile: "tx.pb.go", fdVarName: "fileDescriptor_anchor_tx"},
	{dir: "x/sigil/types", pbFile: "tx.pb.go", fdVarName: "fileDescriptor_sigil_tx"},
	// Query descriptors
	{dir: "x/capability/types", pbFile: "query.pb.go", fdVarName: "fileDescriptor_76eec79927870477"},
	{dir: "x/anchor/types", pbFile: "query.pb.go", fdVarName: "fileDescriptor_anchor_query"},
}

// Known message definitions for types that exist in Go code but not in file descriptors.
var challengeMessages = map[string]*descriptorpb.DescriptorProto{
	// Query messages for Invocation (capability module)
	"QueryInvocationRequest": {
		Name: proto.String("QueryInvocationRequest"),
		Field: []*descriptorpb.FieldDescriptorProto{
			stringField("invocation_id", 1),
		},
	},
	"QueryInvocationResponse": {
		Name: proto.String("QueryInvocationResponse"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:     proto.String("invocation"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".oasyce.capability.v1.Invocation"),
				JsonName: proto.String("invocation"),
			},
		},
	},
	// Challenge window Tx messages
	"MsgCompleteInvocation": {
		Name: proto.String("MsgCompleteInvocation"),
		Field: []*descriptorpb.FieldDescriptorProto{
			stringField("creator", 1),
			stringField("invocation_id", 2),
			stringField("output_hash", 3),
			stringField("usage_report", 4),
		},
	},
	"MsgCompleteInvocationResponse": emptyMessage("MsgCompleteInvocationResponse"),
	"MsgFailInvocation": {
		Name: proto.String("MsgFailInvocation"),
		Field: []*descriptorpb.FieldDescriptorProto{
			stringField("creator", 1),
			stringField("invocation_id", 2),
		},
	},
	"MsgFailInvocationResponse": emptyMessage("MsgFailInvocationResponse"),
	"MsgClaimInvocation": {
		Name: proto.String("MsgClaimInvocation"),
		Field: []*descriptorpb.FieldDescriptorProto{
			stringField("creator", 1),
			stringField("invocation_id", 2),
		},
	},
	"MsgClaimInvocationResponse": emptyMessage("MsgClaimInvocationResponse"),
	"MsgDisputeInvocation": {
		Name: proto.String("MsgDisputeInvocation"),
		Field: []*descriptorpb.FieldDescriptorProto{
			stringField("creator", 1),
			stringField("invocation_id", 2),
			stringField("reason", 3),
		},
	},
	"MsgDisputeInvocationResponse": emptyMessage("MsgDisputeInvocationResponse"),
	// Datarights: MsgUpdateServiceUrl (hand-written, Descriptor() returns nil)
	"MsgUpdateServiceUrl": {
		Name: proto.String("MsgUpdateServiceUrl"),
		Field: []*descriptorpb.FieldDescriptorProto{
			stringField("creator", 1),
			stringField("asset_id", 2),
			stringField("service_url", 3),
		},
	},
	"MsgUpdateServiceUrlResponse": emptyMessage("MsgUpdateServiceUrlResponse"),
	// Anchor query: AnchorsBySigil
	"QueryAnchorsBySigilRequest": {
		Name: proto.String("QueryAnchorsBySigilRequest"),
		Field: []*descriptorpb.FieldDescriptorProto{
			stringField("sigil_id", 1),
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
	"QueryAnchorsBySigilResponse": {
		Name: proto.String("QueryAnchorsBySigilResponse"),
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
}

func addPulseMessages(fd *descriptorpb.FileDescriptorProto, pkg string) {
	fd.MessageType = append(fd.MessageType,
		&descriptorpb.DescriptorProto{
			Name: proto.String("MsgPulse"),
			Field: []*descriptorpb.FieldDescriptorProto{
				stringField("signer", 1),
				stringField("sigil_id", 2),
				{
					Name:     proto.String("dimensions"),
					Number:   proto.Int32(3),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String("." + pkg + ".MsgPulse.DimensionsEntry"),
					JsonName: proto.String("dimensions"),
				},
			},
			NestedType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("DimensionsEntry"),
					Field: []*descriptorpb.FieldDescriptorProto{
						stringField("key", 1),
						{
							Name:     proto.String("value"),
							Number:   proto.Int32(2),
							Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
							JsonName: proto.String("value"),
						},
					},
					Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
				},
			},
		},
		emptyMessage("MsgPulseResponse"),
	)
}

func stringField(name string, num int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(num),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		JsonName: proto.String(toJsonName(name)),
	}
}

func toJsonName(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func emptyMessage(name string) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{Name: proto.String(name)}
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	for _, m := range modules {
		fmt.Printf("Patching %s ...\n", m.dir)
		if err := patchModule(root, m); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR %s: %v\n", m.dir, err)
			os.Exit(1)
		}
	}

	// Post-patch validation: verify all messages have Descriptor() and signer options
	fmt.Println("\nValidating...")
	hasErrors := false
	for _, m := range modules {
		if errs := validateModule(root, m); len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "VALIDATION ERROR %s: %s\n", m.dir, e)
			}
			hasErrors = true
		}
	}
	if hasErrors {
		fmt.Fprintf(os.Stderr, "\nValidation failed! Fix the issues above.\n")
		os.Exit(1)
	}
	fmt.Println("Validation passed.")
	fmt.Println("Done.")
}

func patchModule(root string, m moduleInfo) error {
	pbPath := filepath.Join(root, m.dir, m.pbFile)
	data, err := os.ReadFile(pbPath)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	content := string(data)

	isQuery := strings.HasSuffix(m.pbFile, "query.pb.go")

	// Discover which RPC methods exist in the Go service descriptor
	goMethods := extractGoServiceMethods(content, isQuery)
	fmt.Printf("  Go service methods: %v\n", goMethods)

	// Extract and parse file descriptor
	fdBytes, err := extractFileDescriptorBytes(content, m.fdVarName)
	if err != nil {
		return fmt.Errorf("extract fd: %w", err)
	}
	rawFD, err := decompressGzip(fdBytes)
	if err != nil {
		return fmt.Errorf("decompress: %w", err)
	}
	var fd descriptorpb.FileDescriptorProto
	if err := proto.Unmarshal(rawFD, &fd); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	pkg := fd.GetPackage()
	fmt.Printf("  Package: %s, Messages: %d, Services: %d\n", pkg, len(fd.GetMessageType()), len(fd.GetService()))

	// Collect existing proto RPC method names
	existingRPCs := map[string]bool{}
	if len(fd.GetService()) > 0 {
		for _, m := range fd.Service[0].GetMethod() {
			existingRPCs[m.GetName()] = true
		}
	}
	existingMsgs := map[string]bool{}
	for _, msg := range fd.GetMessageType() {
		existingMsgs[msg.GetName()] = true
	}

	changed := false

	// Add missing RPC methods + message types
	for _, method := range goMethods {
		// Naming convention: tx -> Msg{Method}/Msg{Method}Response, query -> Query{Method}Request/Query{Method}Response
		var reqName, respName string
		if isQuery {
			reqName = "Query" + method + "Request"
			respName = "Query" + method + "Response"
		} else {
			reqName = "Msg" + method
			respName = "Msg" + method + "Response"
		}

		// Add request message if missing (even if RPC already exists)
		if !existingMsgs[reqName] {
			if def, ok := challengeMessages[reqName]; ok {
				fd.MessageType = append(fd.MessageType, def)
			} else if method == "Pulse" {
				addPulseMessages(&fd, pkg)
				existingMsgs[reqName] = true
				existingMsgs[respName] = true
				changed = true
				fmt.Printf("  Added messages: %s, %s\n", reqName, respName)
			} else if method == "UpdateParams" {
				addUpdateParamsMessages(&fd, pkg)
			} else {
				fmt.Printf("  WARNING: no message definition for %s\n", reqName)
				if existingRPCs[method] {
					continue
				}
				continue
			}
			existingMsgs[reqName] = true
			changed = true
			fmt.Printf("  Added message: %s\n", reqName)
		}
		// Add response message if missing
		if !existingMsgs[respName] {
			if def, ok := challengeMessages[respName]; ok {
				fd.MessageType = append(fd.MessageType, def)
			} else if method == "UpdateParams" {
				// Already added by addUpdateParamsMessages
			} else {
				fd.MessageType = append(fd.MessageType, emptyMessage(respName))
			}
			existingMsgs[respName] = true
			changed = true
			fmt.Printf("  Added message: %s\n", respName)
		}

		// Add RPC method if missing
		if existingRPCs[method] {
			continue
		}
		rpc := &descriptorpb.MethodDescriptorProto{
			Name:       proto.String(method),
			InputType:  proto.String("." + pkg + "." + reqName),
			OutputType: proto.String("." + pkg + "." + respName),
		}
		fd.Service[0].Method = append(fd.Service[0].Method, rpc)
		changed = true
		fmt.Printf("  Added RPC: %s\n", method)
	}

	// Ensure genesis.proto is imported (for Params type in UpdateParams)
	if hasMethod(goMethods, "UpdateParams") {
		addGenesisImport(&fd)
		changed = true
	}

	// Ensure all Msg messages have cosmos.msg.v1.signer option
	if !isQuery {
		if ensureSignerOptions(&fd, pkg) {
			changed = true
		}
	}

	if !changed {
		fmt.Printf("  No changes needed.\n")
		return nil
	}

	// Re-marshal and compress
	newRaw, err := proto.Marshal(&fd)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	newCompressed, err := compressGzip(newRaw)
	if err != nil {
		return fmt.Errorf("compress: %w", err)
	}

	goBytes := formatGoBytes(newCompressed, len(newCompressed))
	newContent, err := replaceFileDescriptor(content, m.fdVarName, goBytes)
	if err != nil {
		return fmt.Errorf("replace: %w", err)
	}

	if err := os.WriteFile(pbPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	fmt.Printf("  Written %s (%d → %d bytes compressed)\n", pbPath, len(fdBytes), len(newCompressed))
	return nil
}

// validateModule checks that every message type in the file descriptor has:
// 1. A Descriptor() method in the Go source
// 2. cosmos.msg.v1.signer option for Msg* request types (non-Response, non-Query)
func validateModule(root string, m moduleInfo) []string {
	var errs []string

	// Read the pb file and parse file descriptor
	pbPath := filepath.Join(root, m.dir, m.pbFile)
	data, err := os.ReadFile(pbPath)
	if err != nil {
		return []string{fmt.Sprintf("read %s: %v", pbPath, err)}
	}
	content := string(data)

	fdBytes, err := extractFileDescriptorBytes(content, m.fdVarName)
	if err != nil {
		return []string{fmt.Sprintf("extract fd: %v", err)}
	}
	rawFD, err := decompressGzip(fdBytes)
	if err != nil {
		return []string{fmt.Sprintf("decompress: %v", err)}
	}
	var fd descriptorpb.FileDescriptorProto
	if err := proto.Unmarshal(rawFD, &fd); err != nil {
		return []string{fmt.Sprintf("unmarshal: %v", err)}
	}

	// Read ALL .go files in the types directory to find Descriptor() methods
	typesDir := filepath.Join(root, m.dir)
	goFiles, _ := filepath.Glob(filepath.Join(typesDir, "*.go"))
	var allGoSource string
	for _, f := range goFiles {
		src, _ := os.ReadFile(f)
		allGoSource += string(src)
	}

	// Check 1: Every message must have a Descriptor() method somewhere in the Go source
	// Match only actual method declarations (line must start with func, not be commented out)
	descriptorRe := regexp.MustCompile(`(?m)^func \(\*(\w+)\) Descriptor\(\)`)
	descriptorTypes := map[string]bool{}
	for _, match := range descriptorRe.FindAllStringSubmatch(allGoSource, -1) {
		descriptorTypes[match[1]] = true
	}

	isQuery := strings.HasSuffix(m.pbFile, "query.pb.go")
	for _, msg := range fd.GetMessageType() {
		name := msg.GetName()
		if !descriptorTypes[name] {
			errs = append(errs, fmt.Sprintf("message %s has no Descriptor() method in Go source", name))
		}
	}

	// Check 2: Every Msg* request type (not Response, not Query) must have signer option
	if !isQuery {
		for _, msg := range fd.GetMessageType() {
			name := msg.GetName()
			if !strings.HasPrefix(name, "Msg") || strings.HasSuffix(name, "Response") {
				continue
			}
			opts := msg.GetOptions()
			if opts == nil {
				errs = append(errs, fmt.Sprintf("message %s has no cosmos.msg.v1.signer option", name))
				continue
			}
			raw, _ := proto.Marshal(opts)
			if len(raw) == 0 {
				errs = append(errs, fmt.Sprintf("message %s has empty options (missing signer)", name))
			}
		}
	}

	return errs
}

func hasMethod(methods []string, name string) bool {
	for _, m := range methods {
		if m == name {
			return true
		}
	}
	return false
}

// extractGoServiceMethods parses the service descriptor in Go code to find method names.
var methodNameRe = regexp.MustCompile(`MethodName:\s*"(\w+)"`)

func extractGoServiceMethods(content string, isQuery bool) []string {
	var marker string
	if isQuery {
		marker = "_Query_serviceDesc = grpc.ServiceDesc{"
	} else {
		marker = "_Msg_serviceDesc = grpc.ServiceDesc{"
	}
	idx := strings.Index(content, marker)
	if idx == -1 {
		return nil
	}
	end := strings.Index(content[idx:], "Streams:")
	if end == -1 {
		return nil
	}
	section := content[idx : idx+end]

	matches := methodNameRe.FindAllStringSubmatch(section, -1)
	var result []string
	for _, m := range matches {
		result = append(result, m[1])
	}
	return result
}

func addUpdateParamsMessages(fd *descriptorpb.FileDescriptorProto, pkg string) {
	fd.MessageType = append(fd.MessageType,
		&descriptorpb.DescriptorProto{
			Name: proto.String("MsgUpdateParams"),
			Field: []*descriptorpb.FieldDescriptorProto{
				stringField("authority", 1),
				{
					Name:     proto.String("params"),
					Number:   proto.Int32(2),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String("." + pkg + ".Params"),
					JsonName: proto.String("params"),
				},
			},
		},
		emptyMessage("MsgUpdateParamsResponse"),
	)
}

// ensureSignerOptions adds cosmos.msg.v1.signer option to Msg messages that lack it.
// Extension field 11110000 = cosmos.msg.v1.signer (string, LDel)
// Extension field 11110001 = amino type URL (string, LDel)
func ensureSignerOptions(fd *descriptorpb.FileDescriptorProto, pkg string) bool {
	changed := false
	for _, msg := range fd.GetMessageType() {
		name := msg.GetName()
		// Only add options to request messages (Msg*), not responses
		if !strings.HasPrefix(name, "Msg") || strings.HasSuffix(name, "Response") {
			continue
		}
		// Check if options already exist
		if msg.GetOptions() != nil {
			raw, _ := proto.Marshal(msg.GetOptions())
			if len(raw) > 0 {
				continue
			}
		}
		// Determine signer field name: prefer "authority", then "signer", default "creator"
		signerField := "creator"
		for _, f := range msg.GetField() {
			switch f.GetName() {
			case "authority":
				signerField = "authority"
			case "signer":
				signerField = "signer"
			}
		}
		// Build amino type URL: e.g. "oasyce/capability/MsgUpdateParams"
		parts := strings.Split(pkg, ".")
		moduleName := parts[1] // oasyce.capability.v1 -> capability
		aminoURL := "oasyce/" + moduleName + "/" + name

		// Build raw options bytes with extension fields
		opts := buildSignerOptions(signerField, aminoURL)
		msg.Options = &descriptorpb.MessageOptions{}
		msg.Options.ProtoReflect().SetUnknown(opts)
		changed = true
		fmt.Printf("  Added signer option (%s) to %s\n", signerField, name)
	}
	return changed
}

// buildSignerOptions constructs raw protobuf bytes for cosmos.msg.v1.signer and amino options.
func buildSignerOptions(signer, aminoURL string) []byte {
	var buf []byte
	// Field 11110000 (cosmos.msg.v1.signer), wire type 2 (LDel)
	buf = appendVarint(buf, (11110000<<3)|2)
	buf = appendVarint(buf, uint64(len(signer)))
	buf = append(buf, signer...)
	// Field 11110001 (amino type URL), wire type 2 (LDel)
	buf = appendVarint(buf, (11110001<<3)|2)
	buf = appendVarint(buf, uint64(len(aminoURL)))
	buf = append(buf, aminoURL...)
	return buf
}

func appendVarint(buf []byte, x uint64) []byte {
	for x >= 0x80 {
		buf = append(buf, byte(x)|0x80)
		x >>= 7
	}
	return append(buf, byte(x))
}

func addGenesisImport(fd *descriptorpb.FileDescriptorProto) {
	txProto := fd.GetName()
	genesisProto := strings.Replace(txProto, "/tx.proto", "/genesis.proto", 1)
	for _, dep := range fd.GetDependency() {
		if dep == genesisProto {
			return
		}
	}
	fd.Dependency = append(fd.Dependency, genesisProto)
}

func extractFileDescriptorBytes(content, varName string) ([]byte, error) {
	marker := "var " + varName + " = []byte{"
	idx := strings.Index(content, marker)
	if idx == -1 {
		return nil, fmt.Errorf("variable %s not found", varName)
	}
	start := idx + len(marker)
	end := strings.Index(content[start:], "}")
	if end == -1 {
		return nil, fmt.Errorf("closing brace not found")
	}
	byteStr := content[start : start+end]

	var result []byte
	for _, line := range strings.Split(byteStr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		for _, part := range strings.Split(line, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			var b byte
			if _, err := fmt.Sscanf(part, "0x%x", &b); err == nil {
				result = append(result, b)
			}
		}
	}
	return result, nil
}

func decompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatGoBytes(data []byte, totalLen int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\t// %d bytes of a gzipped FileDescriptorProto\n", totalLen))
	for i := 0; i < len(data); i += 16 {
		sb.WriteString("\t")
		end := i + 16
		if end > len(data) {
			end = len(data)
		}
		for j := i; j < end; j++ {
			if j > i {
				sb.WriteString(" ")
			}
			sb.WriteString(fmt.Sprintf("0x%02x,", data[j]))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func replaceFileDescriptor(content, varName, newBytes string) (string, error) {
	marker := "var " + varName + " = []byte{"
	idx := strings.Index(content, marker)
	if idx == -1 {
		return "", fmt.Errorf("variable %s not found", varName)
	}
	end := strings.Index(content[idx:], "}\n")
	if end == -1 {
		return "", fmt.Errorf("closing brace not found")
	}
	replacement := marker + "\n" + newBytes
	return content[:idx] + replacement + content[idx+end:], nil
}
