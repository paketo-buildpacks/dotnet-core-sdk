package sdk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/dotnet-core-conf-cnb/utils"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/mitchellh/mapstructure"
)

const DotnetSDK = "dotnet-sdk"

type Contributor struct {
	context         build.Build
	plan            buildpackplan.Plan
	sdkLayer        layers.DependencyLayer
	sdkSymlinkLayer layers.Layer
	logger          logger.Logger
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	plan, wantDependency, err := context.Plans.GetShallowMerged(DotnetSDK)
	if err != nil {
		return Contributor{}, false, err
	}
	if !wantDependency {
		return Contributor{}, false, nil
	}

	version := plan.Version

	runtimetoSDK, err := GetRuntimetoSDKMap(context)
	if err != nil {
		return Contributor{}, false, err
	}

	if version != "" {
		floatVersion, err := GetSDKFloatVersion(version)
		if err != nil {
			return Contributor{}, false, err
		}
		compatibleDeps, err := GetLatestCompatibleSDKDeps(floatVersion, context)
		if err != nil {
			return Contributor{}, false, err
		}

		buildpackYAMLVersion, err := loadBuildpackYAMLVersion(context.Application.Root)
		if err != nil {
			return Contributor{}, false, err
		}

		globalJSONVersion, err := loadGlobalJSONVersion(context.Application.Root)
		if err != nil {
			return Contributor{}, false, err
		}

		useGlobalJSON := globalJSONVersion != ""
		useBuildpackYAML := buildpackYAMLVersion != ""

		if useBuildpackYAML && useGlobalJSON {
			useBuildpackYAML, useGlobalJSON, err = SelectRollStrategy(buildpackYAMLVersion, globalJSONVersion)
			if err != nil {
				return Contributor{}, false, err
			}
		}

		var rollForwardError error
		if useBuildpackYAML {
			version, rollForwardError = GetConstrainedCompatibleSDK(buildpackYAMLVersion, runtimetoSDK, compatibleDeps)
		} else if useGlobalJSON {
			version, rollForwardError = GetConstrainedCompatibleSDKForGlobalJson(globalJSONVersion, compatibleDeps)
		} else {
			version, rollForwardError = GetConstrainedCompatibleSDK(floatVersion, runtimetoSDK, compatibleDeps)
		}
		if rollForwardError != nil {
			return Contributor{}, false, err
		}
	}

	dep, err := context.Buildpack.RuntimeDependency(DotnetSDK, version, context.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	return Contributor{
		context:         context,
		plan:            plan,
		sdkLayer:        context.Layers.DependencyLayer(dep),
		sdkSymlinkLayer: context.Layers.Layer("driver-symlinks"),
		logger:          context.Logger,
	}, true, nil
}

func (c Contributor) Contribute() error {

	contributedSDK := false
	err := c.sdkLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.Body("Expanding to %s", layer.Root)
		contributedSDK = true
		if err := helper.ExtractTarXz(artifact, layer.Root, 0); err != nil {
			return err
		}

		if err := layer.OverrideBuildEnv("SDK_LOCATION", layer.Root); err != nil {
			return err
		}

		return nil
	}, layers.Build)

	if err != nil {
		return err
	}

	err = c.sdkSymlinkLayer.Contribute(c.context.Buildpack, func(layer layers.Layer) error {
		if !contributedSDK {
			return nil
		}

		layer.Logger.Body("Symlinking runtime libraries")
		pathToRuntime := os.Getenv("DOTNET_ROOT")

		if err := utils.SymlinkSharedFolder(pathToRuntime, layer.Root); err != nil {
			return err
		}

		hostDir := filepath.Join(pathToRuntime, "host")

		if err := utils.CreateValidSymlink(hostDir, filepath.Join(layer.Root, filepath.Base(hostDir))); err != nil {
			return err
		}
		layer.Logger.Body("Moving dotnet driver from %s", c.sdkLayer.Root)

		if err := helper.CopyFile(filepath.Join(c.sdkLayer.Root, "dotnet"), filepath.Join(layer.Root, "dotnet")); err != nil {
			return err
		}

		if err := layer.AppendPathSharedEnv("PATH", layer.Root); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("DOTNET_ROOT", layer.Root); err != nil {
			return err
		}

		return nil
	}, getFlags(c.plan.Metadata)...)

	if err != nil {
		return err
	}

	return nil
}

func getFlags(metadata buildpackplan.Metadata) []layers.Flag {
	flagsArray := []layers.Flag{}
	flagValueMap := map[string]layers.Flag{"build": layers.Build, "launch": layers.Launch, "cache": layers.Cache}
	for _, flagName := range []string{"build", "launch", "cache"} {
		flagPresent, _ := metadata[flagName].(bool)
		if flagPresent {
			flagsArray = append(flagsArray, flagValueMap[flagName])
		}
	}
	return flagsArray
}

func loadBuildpackYAMLVersion(appRoot string) (string, error) {
	type buildpackYAML struct {
		Config struct {
			Version string `yaml:"version"`
		} `yaml:"dotnet-sdk"`
	}
	var err error
	bpYAML := buildpackYAML{}
	bpYamlPath := filepath.Join(appRoot, "buildpack.yml")

	if exists, err := helper.FileExists(bpYamlPath); err != nil {
		return "", err
	} else if exists {
		err = helper.ReadBuildpackYaml(bpYamlPath, &bpYAML)
	}

	if bpYAML == (buildpackYAML{}) {
		return "", err
	}
	return bpYAML.Config.Version, err
}

func loadGlobalJSONVersion(appRoot string) (string, error) {
	type globalJSONLoad struct {
		Sdk struct {
			Version string `json:"version"`
		} `json:"sdk"`
	}

	var err error
	globalJsonPath := filepath.Join(appRoot, "global.json")
	globalJson := globalJSONLoad{}

	if exists, err := helper.FileExists(globalJsonPath); err != nil {
		return "", err
	} else if exists {
		f, err := os.Open(globalJsonPath)
		if err != nil {
			return "", err
		}

		jsonDecoder := json.NewDecoder(f)
		jsonDecoder.Decode(&globalJson)
	}
	return globalJson.Sdk.Version, err
}

func GetRuntimetoSDKMap(context build.Build) (map[string][]string, error) {
	runtimetoSDK := map[string][]string{}
	type sdkMap struct {
		RuntimeVersion string   `mapstructure:"runtime-version"`
		Sdks           []string `mapstructure:"sdks"`
	}

	metadata, ok := context.Buildpack.Metadata["runtime-to-sdks"].([]map[string]interface{})
	if !ok {
		return runtimetoSDK, fmt.Errorf("unexpected metadata format for sdk to runtime mapping")
	}

	var sdkMapping []sdkMap
	mapstructure.Decode(metadata, &sdkMapping)

	for _, runtimeMapping := range sdkMapping {
		runtimetoSDK[runtimeMapping.RuntimeVersion] = runtimeMapping.Sdks
	}

	return runtimetoSDK, nil
}
