package utils

import (
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/helper"
)

func GetAppRoot(appRoot string) (string, error) {
	type buildpackYAML struct {
		Config struct {
			ProjectPath string `yaml:"project-path"`
		} `yaml:"dotnet-build"`
	}
	var bpYAML buildpackYAML
	bpYamlPath := filepath.Join(appRoot, "buildpack.yml")

	exists, err := helper.FileExists(bpYamlPath)
	if err != nil {
		return appRoot, err
	}

	if exists {
		err = helper.ReadBuildpackYaml(bpYamlPath, &bpYAML)
		if err != nil {
			return appRoot, err
		}
	}

	if bpYAML.Config.ProjectPath != "" {
		return filepath.Join(appRoot, bpYAML.Config.ProjectPath), nil
	}

	return appRoot, nil
}
