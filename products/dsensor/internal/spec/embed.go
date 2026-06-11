package spec

import "embed"

// SpecFS embeds the compiled OpenAPI JSON spec into the binary.
//
//go:embed openapi.json
var SpecFS embed.FS

// SpecJSON returns the raw embedded spec bytes, loading once and caching.
var SpecJSON []byte

func init() {
	var err error
	SpecJSON, err = SpecFS.ReadFile("openapi.json")
	if err != nil {
		panic("failed to read embedded openapi.json: " + err.Error())
	}
}
