package dotnetcoresdk

import (
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/fs"
)

type Symlinker struct{}

func NewSymlinker() Symlinker {
	return Symlinker{}
}

func (s Symlinker) Link(workingDir, layerPath string) error {
	dirs := []string{"sdk", "templates", "packs"}

	var err error
	for _, dir := range dirs {
		err = createAndLink(workingDir, layerPath, dir)
		if err != nil {
			return err
		}
	}

	err = fs.Copy(filepath.Join(layerPath, "dotnet"), filepath.Join(workingDir, ".dotnet_root", "dotnet"))
	if err != nil {
		return err
	}

	return nil
}

func createAndLink(workingDir, layerPath, dirName string) error {
	err := os.MkdirAll(filepath.Join(workingDir, ".dotnet_root"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, dirName), filepath.Join(workingDir, ".dotnet_root", dirName))
	if err != nil {
		return err
	}

	return nil

}
