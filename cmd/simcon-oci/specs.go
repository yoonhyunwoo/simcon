package main

import (
	"encoding/json"
	"github.com/opencontainers/runtime-spec/specs-go"
	"os"
)

func loadConfig(cPath string) (*specs.Spec, error) {
	cData, err := os.Open(cPath)
	if err != nil {
		return nil, err
	}
	defer cData.Close()

	config := &specs.Spec{}
	if err := json.NewDecoder(cData).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
