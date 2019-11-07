package aspnet

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/dotnet-core-conf-cnb/utils"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
)

const DotnetAspNet = "dotnet-aspnetcore"

type Contributor struct {
	context            build.Build
	plan               buildpackplan.Plan
	aspnetLayer        layers.DependencyLayer
	aspnetSymlinkLayer layers.Layer
	logger             logger.Logger
}

type BuildpackYAML struct {
	Config struct {
		Version string `yaml:"version""`
	} `yaml:"dotnet-framework"`
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	plan, wantDependency, err := context.Plans.GetShallowMerged(DotnetAspNet)
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
			version, err = utils.FrameworkRollForward(rollForwardVersion, DotnetAspNet, context)
			if err != nil {
				return Contributor{}, false, err
			}
		}

		if version == "" {
			return Contributor{}, false, fmt.Errorf("no version of the dotnet-runtime was compatible with what was specified in the runtimeconfig.json of the application")
		}
	}

	dep, err := context.Buildpack.RuntimeDependency(DotnetAspNet, version, context.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	return Contributor{
		context:            context,
		plan:               plan,
		aspnetLayer:        context.Layers.DependencyLayer(dep),
		aspnetSymlinkLayer: context.Layers.Layer("aspnet-symlinks"),
		logger:             context.Logger,
	}, true, nil
}

func (c Contributor) Contribute() error {
	err := c.aspnetLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.Body("Expanding to %s", layer.Root)

		if err := helper.ExtractTarXz(artifact, layer.Root, 0); err != nil {
			return err
		}

		return nil
	}, getFlags(c.plan.Metadata)...)

	if err != nil {
		return err
	}

	err = c.aspnetSymlinkLayer.Contribute(c.context.Buildpack, func(layer layers.Layer) error {
		pathToRuntime := os.Getenv("DOTNET_ROOT")

		if err := utils.SymlinkSharedFolder(pathToRuntime, layer.Root); err != nil {
			return err
		}

		if err := utils.SymlinkSharedFolder(c.aspnetLayer.Root, layer.Root); err != nil {
			return err
		}

		hostDir := filepath.Join(pathToRuntime, "host")

		if err := helper.WriteSymlink(hostDir, filepath.Join(layer.Root, filepath.Base(hostDir))); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("DOTNET_ROOT", filepath.Join(layer.Root)); err != nil {
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
