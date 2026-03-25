// +build ignore

// Quick script to check which RPCs are in query file descriptors.
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

var queryFiles = []struct {
	path    string
	fdVar   string
}{
	{"x/capability/types/query.pb.go", "fileDescriptor_76eec79927870477"},
	{"x/datarights/types/query.pb.go", "fileDescriptor_5861e7fe503b6617"},
	{"x/onboarding/types/query.pb.go", "fileDescriptor_54be95d4d3143d75"},
	{"x/reputation/types/query.pb.go", "fileDescriptor_d9c7271d83d7f7aa"},
	{"x/settlement/types/query.pb.go", "fileDescriptor_be10005f37ec133f"},
	{"x/work/types/query.pb.go", "fileDescriptor_1e7a7341d0bef616"},
}

func main() {
	for _, qf := range queryFiles {
		data, err := os.ReadFile(qf.path)
		if err != nil {
			fmt.Printf("ERROR reading %s: %v\n", qf.path, err)
			continue
		}
		content := string(data)

		fdBytes, err := extractFDBytes(content, qf.fdVar)
		if err != nil {
			fmt.Printf("ERROR extracting %s: %v\n", qf.path, err)
			continue
		}

		r, _ := gzip.NewReader(bytes.NewReader(fdBytes))
		raw, _ := io.ReadAll(r)
		r.Close()

		var fd descriptorpb.FileDescriptorProto
		proto.Unmarshal(raw, &fd)

		fmt.Printf("%s:\n", qf.path)
		fmt.Printf("  Messages: %d\n", len(fd.GetMessageType()))
		for i, msg := range fd.GetMessageType() {
			fmt.Printf("    [%d] %s\n", i, msg.GetName())
		}
		if len(fd.GetService()) > 0 {
			svc := fd.GetService()[0]
			fmt.Printf("  Service RPCs: %d\n", len(svc.GetMethod()))
			for _, m := range svc.GetMethod() {
				fmt.Printf("    %s\n", m.GetName())
			}
		}
		fmt.Println()
	}
}

func extractFDBytes(content, varName string) ([]byte, error) {
	marker := "var " + varName + " = []byte{"
	idx := strings.Index(content, marker)
	if idx == -1 {
		return nil, fmt.Errorf("not found")
	}
	start := idx + len(marker)
	end := strings.Index(content[start:], "}")
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
