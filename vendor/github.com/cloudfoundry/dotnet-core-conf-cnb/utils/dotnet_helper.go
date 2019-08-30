package utils

import (
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/helper"
)


func CreateValidSymlink(oldName string, newName string) error {
	if exists, err := helper.FileExists(oldName); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("cannot create invalid symlink, %s does not exits", oldName)
	}
	return helper.WriteSymlink(oldName, newName)
}
