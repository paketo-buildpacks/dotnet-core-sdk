package dotnetcoresdk

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type SdkVersionParser struct{}

func NewSdkVersionParser() SdkVersionParser {
	return SdkVersionParser{}
}

func (p SdkVersionParser) Parse(workingDir string) (string, error) {
	file, err := os.Open(filepath.Join(workingDir, "buildpack.yml"))
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("could not open buildpack.yml: %s", err)
		}
		return "", nil
	}

	var data struct {
		DotnetSdk struct {
			Version string `yaml:"version"`
		} `yaml:"dotnet-sdk"`
	}
	err = yaml.NewDecoder(file).Decode(&data)
	if err != nil {
		return "", err
	}
	return data.DotnetSdk.Version, nil
}
