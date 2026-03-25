package docs

import (
	_ "embed"
)

// OpenAPISpec is the embedded OpenAPI specification served at /openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPISpec []byte

// LLMSTxt is the embedded llms.txt served at /llms.txt.
// Kept in sync with the root llms.txt via build process.
//
//go:embed llms.txt
var LLMSTxt []byte

// ErrorCodes is the embedded error code catalog served at /oasyce/v1/error-codes.
//
//go:embed error_codes.json
var ErrorCodes []byte
