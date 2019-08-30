package sdk

import (
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

func NewContributor(context build.Build) (Contributor, bool, error) {
	plan, wantDependency, err := context.Plans.GetShallowMerged(DotnetSDK)
	if err != nil {
		return Contributor{}, false, err
	}
	if !wantDependency {
		return Contributor{}, false, nil
	}

	dep, err := context.Buildpack.RuntimeDependency(DotnetSDK, plan.Version, context.Stack)
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

	err := c.sdkLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.Body("Expanding to %s", layer.Root)

		if err := helper.ExtractTarXz(artifact, layer.Root, 0); err != nil {
			return err
		}

		return nil
	}, getFlags(c.plan.Metadata)...)

	err = c.sdkSymlinkLayer.Contribute(c.context.Buildpack, func(layer layers.Layer) error {
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

		// make new symlink for host from sdk layer
		hostDir := filepath.Join(c.sdkLayer.Root, "host")

		if err := utils.CreateValidSymlink(hostDir, filepath.Join(layer.Root, filepath.Base(hostDir))); err != nil {
			return err
		}
		layer.Logger.Body("Moving dotnet driver from %s", c.sdkLayer.Root)


		// copy dotnet file from sdk
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

