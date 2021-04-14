package dotnetcoresdk

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver"
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

//go:generate faux --interface DependencyMapper --output fakes/dependency_mapper.go
type DependencyMapper interface {
	FindCorrespondingVersion(path, versionKey string) (string, error)
}

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Resolve(path, id, version, stack string) (postal.Dependency, error)
	Install(dependency postal.Dependency, cnbPath, layerPath string) error
}

//go:generate faux --interface DotnetSymlinker --output fakes/dotnet_symlinker.go
type DotnetSymlinker interface {
	Link(workingDir, layerPath string) error
}

func Build(entryResolver EntryResolver,
	dependencyMapper DependencyMapper,
	buildPlanRefinery BuildPlanRefinery,
	dependencyManager DependencyManager,
	dotnetSymlinker DotnetSymlinker,
	logger LogEmitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)
		logger.Process("Resolving .NET Core SDK version")

		if runtimeVersion, ok := os.LookupEnv("RUNTIME_VERSION"); ok {
			sdkVersion, err := dependencyMapper.FindCorrespondingVersion(filepath.Join(context.CNBPath, "buildpack.toml"), runtimeVersion)
			if err != nil {
				return packit.BuildResult{}, err
			}

			context.Plan.Entries = append(context.Plan.Entries, packit.BuildpackPlanEntry{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": "RUNTIME_VERSION",
					"version":        sdkVersion,
				},
			})
		}

		planEntry := entryResolver.Resolve(context.Plan.Entries)
		version, _ := planEntry.Metadata["version"].(string)
		versionSource, _ := planEntry.Metadata["version-source"].(string)

		sdkDependency, err := dependencyManager.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), planEntry.Name, version, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(planEntry, sdkDependency, clock.Now())

		if versionSource == "buildpack.yml" {
			nextMajorVersion := semver.MustParse(context.BuildpackInfo.Version).IncMajor()
			logger.Break()
			logger.Subprocess("WARNING: Setting the .NET Core SDK version through buildpack.yml will be deprecated soon in Dotnet Core SDK Buildpack v%s.", nextMajorVersion.String())
		}

		bom := buildPlanRefinery.BillOfMaterial(sdkDependency)

		sdkLayer, err := context.Layers.Get("dotnet-core-sdk")
		if err != nil {
			return packit.BuildResult{}, err
		}

		envLayer, err := context.Layers.Get("dotnet-env-var")
		if err != nil {
			return packit.BuildResult{}, err
		}

		envLayer.Launch = true
		envLayer.Build = true

		cachedDependencySHA, ok := sdkLayer.Metadata["dependency-sha"]
		if ok && cachedDependencySHA == sdkDependency.SHA256 {
			logger.Process(fmt.Sprintf("Reusing cached layer %s", sdkLayer.Path))
			logger.Break()

			err = dotnetSymlinker.Link(context.WorkingDir, sdkLayer.Path)
			if err != nil {
				return packit.BuildResult{}, err
			}

			logger.Process("Configuring environment")
			envLayer.SharedEnv.Prepend("PATH",
				filepath.Join(context.WorkingDir, ".dotnet_root"),
				string(os.PathListSeparator))

			envLayer.SharedEnv.Override("DOTNET_ROOT", filepath.Join(context.WorkingDir, ".dotnet_root"))
			logger.Environment(envLayer.SharedEnv)

			return packit.BuildResult{
				Plan: context.Plan,
				Layers: []packit.Layer{
					sdkLayer,
					envLayer,
				},
			}, nil
		}

		logger.Process("Executing build process")

		sdkLayer, err = sdkLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Subprocess("Installing %s %s", ".NET Core SDK", sdkDependency.Version)
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
		envLayer.SharedEnv.Prepend("PATH",
			filepath.Join(context.WorkingDir, ".dotnet_root"),
			string(os.PathListSeparator))

		envLayer.SharedEnv.Override("DOTNET_ROOT", filepath.Join(context.WorkingDir, ".dotnet_root"))
		logger.Environment(envLayer.SharedEnv)

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{bom},
			},
			Layers: []packit.Layer{
				sdkLayer,
				envLayer,
			},
		}, nil
	}
}
