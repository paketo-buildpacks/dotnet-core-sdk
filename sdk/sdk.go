package sdk

import (
	"encoding/json"
	"github.com/cloudfoundry/dotnet-core-conf-cnb/utils"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"os"
	"path/filepath"
)

const DotnetSDK = "dotnet-sdk"

type Contributor struct {
	context         build.Build
	plan            buildpackplan.Plan
	sdkLayer        layers.DependencyLayer
	sdkSymlinkLayer layers.Layer
	logger          logger.Logger
}

type BuildpackYAML struct {
	Config struct {
		Version string `yaml:"version""`
	} `yaml:"dotnet-sdk"`
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

	if version != "" {
		version, err = GetLatestCompatibleSDKConstraint(plan.Version)
		if err != nil {
			return Contributor{}, false, err
		}

		buildpackYAML, err := LoadBuildpackYAML(context.Application.Root)
		if err != nil {
			return Contributor{}, false, err
		}

		globalJSONVersion, err := LoadGlobalJSON(context.Application.Root)
		if err != nil {
			return Contributor{}, false, err
		}

		useGlobalJSON := globalJSONVersion != ""
		useBuildpackYAML := buildpackYAML != (BuildpackYAML{})

		var buildpackYAMLVersion string
		if useBuildpackYAML { buildpackYAMLVersion = buildpackYAML.Config.Version}

		if useBuildpackYAML && useGlobalJSON{
			useBuildpackYAML, useGlobalJSON, err = SelectRollStrategy(buildpackYAMLVersion, globalJSONVersion)
			if err != nil {
				return Contributor{}, false, err
			}
		}

		if useBuildpackYAML{
			compatible, err := IsCompatibleSDKOptionWithRuntime(version, buildpackYAMLVersion)
			if err != nil {
				return Contributor{}, false, err
			}

			if compatible {
				version, err = GetConstrainedCompatibleSDK(buildpackYAMLVersion, context)
				if err != nil {
					return Contributor{}, false, err
				}
			}
		}

		if useGlobalJSON{
			compatible, err := IsCompatibleSDKOptionWithRuntime(version, globalJSONVersion)
			if err != nil {
				return Contributor{}, false, err
			}

			if compatible {
				version, err = GetConstrainedCompatibleSDKForGlobalJson(globalJSONVersion, context)
				if err != nil {
					return Contributor{}, false, err
				}
			}
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
		runtimeFiles, err := filepath.Glob(filepath.Join(pathToRuntime, "shared", "*"))
		if err != nil {
			return err
		}
		for _, file := range runtimeFiles {
			if err := helper.CopySymlink(file, filepath.Join(layer.Root, "shared", filepath.Base(file))); err != nil {
				return err
			}
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

func LoadBuildpackYAML(appRoot string) (BuildpackYAML, error) {
	var err error
	buildpackYAML := BuildpackYAML{}
	bpYamlPath := filepath.Join(appRoot, "buildpack.yml")

	if exists, err := helper.FileExists(bpYamlPath); err != nil {
		return BuildpackYAML{}, err
	} else if exists {
		err = helper.ReadBuildpackYaml(bpYamlPath, &buildpackYAML)
	}
	return buildpackYAML, err
}


func LoadGlobalJSON(appRoot string) (string, error) {
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
