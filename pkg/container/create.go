package container

import (
	"path/filepath"

	"github.com/yoonhyunwoo/simcon/pkg/config"
)

func (container *Container) Create(containerID, bundlePath string) error {
	bundlePath, err := filepath.Abs(bundlePath)
	if err != nil {
		return err
	}

	spec, err := config.LoadConfig(bundlePath)
	if err != nil {
		return err
	}

	return nil
}
