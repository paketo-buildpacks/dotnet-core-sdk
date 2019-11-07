package runtime

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/dotnet-core-conf-cnb/utils"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
)

const DotnetRuntime = "dotnet-runtime"

type Contributor struct {
	context      build.Build
	plan         buildpackplan.Plan
	version      string
	runtimeLayer layers.DependencyLayer
	logger       logger.Logger
}

type BuildpackYAML struct {
	Config struct {
		Version string `yaml:"version"`
	} `yaml:"dotnet-framework"`
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	plan, wantDependency, err := context.Plans.GetShallowMerged(DotnetRuntime)
	if err != nil {
		return Contributor{}, false, err
	}
	if !wantDependency {
		return Contributor{}, false, nil
	}

	version := plan.Version

	if plan.Version != "" {
		rollForwardVersion := plan.Version

		buildpackYAML, err := LoadBuildpackYAML(context.Application.Root)
		if err != nil {
			return Contributor{}, false, err
		}

		if buildpackYAML != (BuildpackYAML{}) {
			err := utils.BuildpackYAMLVersionCheck(rollForwardVersion, buildpackYAML.Config.Version)
			if err != nil {
				return Contributor{}, false, err
			}
			version = buildpackYAML.Config.Version
		} else {
			version, err = utils.FrameworkRollForward(rollForwardVersion, DotnetRuntime, context)
			if err != nil {
				return Contributor{}, false, err
			}
		}

		if version == "" {
			return Contributor{}, false, fmt.Errorf("no version of the dotnet-runtime was compatible with what was specified in the runtimeconfig.json of the application")
		}
	}

	dep, err := context.Buildpack.RuntimeDependency(DotnetRuntime, version, context.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	return Contributor{
		context:      context,
		plan:         plan,
		version:      dep.Version.Version.String(),
		runtimeLayer: context.Layers.DependencyLayer(dep),
		logger:       context.Logger,
	}, true, nil
}

func (c Contributor) Contribute() error {

	return c.runtimeLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.Body("Expanding to %s", layer.Root)

		if err := helper.ExtractTarXz(artifact, layer.Root, 0); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("DOTNET_ROOT", filepath.Join(layer.Root)); err != nil {
			return err
		}

		if err := layer.OverrideBuildEnv("RUNTIME_VERSION", c.version); err != nil {
			return err
		}

		return nil
	}, getFlags(c.plan.Metadata)...)
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
