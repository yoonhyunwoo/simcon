package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

var (
	DataDir        = "/var/lib/simcon"
	ConfigFileName = "config.json"
)

func LoadConfig(bundlePath string) (*specs.Spec, error) {
	spec := &specs.Spec{}
	specFile, err := os.Open(filepath.Join(bundlePath, ConfigFileName))
	if err != nil {
		return nil, err
	}
	if err = json.NewDecoder(specFile).Decode(spec); err != nil {
		return nil, err
	}
	return spec, nil
}
