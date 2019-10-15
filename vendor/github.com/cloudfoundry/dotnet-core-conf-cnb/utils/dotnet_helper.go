package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/helper"
)

func SymlinkSharedFolder(dotnetRoot, layerRoot string) error {
	runtimeFiles, err := filepath.Glob(filepath.Join(dotnetRoot, "shared", "*"))
	if err != nil {
		return err
	}
	for _, file := range runtimeFiles {
		if err := CreateValidSymlink(file, filepath.Join(layerRoot, "shared", filepath.Base(file))); err != nil {
			return err
		}
	}
	return nil
}

func CreateValidSymlink(oldName string, newName string) error {
	if exists, err := helper.FileExists(oldName); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("cannot create invalid symlink, %s does not exits", oldName)
	}

	fileStat, err := os.Lstat(oldName)
	if err != nil {
		return err
	}

	if fileStat.Mode() != os.ModeSymlink {
		return helper.WriteSymlink(oldName, newName)

	}

	return helper.CopySymlink(oldName, newName)
}
