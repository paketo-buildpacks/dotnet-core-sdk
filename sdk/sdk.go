package sdk

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/cloudfoundry/dotnet-core-conf-cnb/utils"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/pkg/errors"
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
		version, err = getLatestCompatibleSDK(plan.Version, context)
		if err != nil {
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

		return nil
	}, getFlags(c.plan.Metadata)...)

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
	}, layers.Build, layers.Launch)

	if err != nil {
		return err
	}

	return nil
}

func getFlags(metadata buildpackplan.Metadata) []layers.Flag{
	flagsArray := []layers.Flag{}
	flagValueMap := map[string]layers.Flag {"build": layers.Build, "launch": layers.Launch, "cache": layers.Cache}
	for _, flagName := range []string{"build", "launch", "cache"} {
		flagPresent, _ := metadata[flagName].(bool)
		if flagPresent {
			flagsArray = append(flagsArray, flagValueMap[flagName])
		}
	}
	return flagsArray
}

func getLatestCompatibleSDK(frameworkVersion string, context build.Build) (string, error){
	splitVersion, err := semver.NewVersion(frameworkVersion)
	if err != nil {
		return "", err
	}

	compatibleVersionConstraint := fmt.Sprintf("%d.%d.*", splitVersion.Major(), splitVersion.Minor())

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return "", err
	}

	compatibleVersion, err := deps.Best(DotnetSDK, compatibleVersionConstraint, context.Stack)

	if err != nil {
		return "",  errors.Wrap(err, "no compatible version of the sdk found")
	}

	return compatibleVersion.Version.Original(), nil
}


