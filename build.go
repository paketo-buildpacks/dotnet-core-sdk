package dotnetcoresdk

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(entries []packit.BuildpackPlanEntry) packit.BuildpackPlanEntry
}

//go:generate faux --interface BuildPlanRefinery --output fakes/build_plan_refinery.go
type BuildPlanRefinery interface {
	BillOfMaterial(dependency postal.Dependency) packit.BuildpackPlanEntry
}

//go:generate faux --interface DependencyResolver --output fakes/dependency_resolver.go
type DependencyResolver interface {
	Resolve(cnbDir string, entry packit.BuildpackPlanEntry, stack string) (postal.Dependency, error)
}

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Install(dependency postal.Dependency, cnbPath, layerPath string) error
}

//go:generate faux --interface DotnetSymlinker --output fakes/dotnet_symlinker.go
type DotnetSymlinker interface {
	Link(workingDir, layerPath string) error
}

func Build(entryResolver EntryResolver,
	dependencyResolver DependencyResolver,
	buildPlanRefinery BuildPlanRefinery,
	dependencyManager DependencyManager,
	dotnetSymlinker DotnetSymlinker,
	logger LogEmitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)
		logger.Process("Resolving .NET Core SDK version")

		planEntry := entryResolver.Resolve(context.Plan.Entries)

		sdkDependency, err := dependencyResolver.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), planEntry, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(planEntry, sdkDependency, clock.Now())

		bom := buildPlanRefinery.BillOfMaterial(sdkDependency)

		sdkLayer, err := context.Layers.Get("dotnet-core-sdk")
		if err != nil {
			return packit.BuildResult{}, err
		}

		cachedDependencySHA, ok := sdkLayer.Metadata["dependency-sha"]
		if ok && cachedDependencySHA == sdkDependency.SHA256 {
			logger.Process(fmt.Sprintf("Reusing cached layer %s", sdkLayer.Path))
			logger.Break()

			err = dotnetSymlinker.Link(context.WorkingDir, sdkLayer.Path)
			if err != nil {
				return packit.BuildResult{}, err
			}

			return packit.BuildResult{
				Plan:   context.Plan,
				Layers: []packit.Layer{sdkLayer},
			}, nil
		}

		logger.Process("Executing build process")

		sdkLayer, err = sdkLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Subprocess("Installing %s %s", sdkDependency.Name, sdkDependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencyManager.Install(sdkDependency, context.CNBPath, sdkLayer.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		sdkLayer.Metadata = map[string]interface{}{
			"dependency-sha": sdkDependency.SHA256,
			"built_at":       clock.Now().Format(time.RFC3339Nano),
		}

		err = dotnetSymlinker.Link(context.WorkingDir, sdkLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		sdkLayer.Build = planEntry.Metadata["build"] == true
		sdkLayer.Cache = planEntry.Metadata["build"] == true || planEntry.Metadata["launch"] == true
		sdkLayer.Launch = planEntry.Metadata["launch"] == true

		logger.Process("Configuring environment")
		sdkLayer.SharedEnv.Prepend("PATH",
			filepath.Join(context.WorkingDir, ".dotnet_root"),
			string(os.PathListSeparator))

		sdkLayer.SharedEnv.Override("DOTNET_ROOT", filepath.Join(context.WorkingDir, ".dotnet_root"))
		logger.Environment(sdkLayer.SharedEnv)

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{bom},
			},
			Layers: []packit.Layer{
				sdkLayer,
			},
		}, nil
	}
}
