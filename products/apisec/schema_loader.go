package apisec

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func parseOpenAPI(data []byte) (*OpenAPI, error) {
	var api OpenAPI
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("parse OpenAPI: %w", err)
	}
	return &api, nil
}

func parseCLIMapping(data []byte) (*CLIMapping, error) {
	var mapping CLIMapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("parse CLI mapping: %w", err)
	}
	return &mapping, nil
}
