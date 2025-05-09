package container

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/yoonhyunwoo/simcon/pkg/config"
)

// Container struct for control
type Container struct {
	ID    string
	Spec  *specs.Spec
	State *specs.State
}

// WriteSpec is Writing Container Spec
func WriteSpec(container Container) error {
	specFile, err := os.OpenFile(filepath.Join(config.DataDir, container.ID), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer specFile.Close()

	if err = json.NewEncoder(specFile).Encode(container.State); err != nil {
		return err
	}

	return nil
}
