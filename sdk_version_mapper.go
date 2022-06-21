package dotnetcoresdk

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type RuntimeToSdks struct {
	RuntimeVersion string   `toml:"runtime-version"`
	SDKs           []string `toml:"sdks"`
}

type SDKVersionMapper struct {
	logger scribe.Emitter
}

func NewSDKVersionMapper(logger scribe.Emitter) SDKVersionMapper {
	return SDKVersionMapper{logger: logger}
}

func (r SDKVersionMapper) FindCorrespondingVersion(path, versionKey string) (string, error) {

	var buildpackTOML struct {
		Metadata struct {
			RuntimeToSdks []RuntimeToSdks `toml:"runtime-to-sdks"`
		} `toml:"metadata"`
	}

	_, err := toml.DecodeFile(path, &buildpackTOML)
	if err != nil {
		return "", fmt.Errorf("buildpack.toml could not be parsed: %w", err)
	}

	runtimeToSDKVersion := map[string]string{}

	for _, mapping := range buildpackTOML.Metadata.RuntimeToSdks {
		runtimeToSDKVersion[mapping.RuntimeVersion] = mapping.SDKs[0]
	}

	if compatibleSDKVersion, ok := runtimeToSDKVersion[versionKey]; ok {
		return compatibleSDKVersion, nil
	}
	return "", fmt.Errorf("no compatible SDK version available for .NET Runtime version %s", versionKey)
}
