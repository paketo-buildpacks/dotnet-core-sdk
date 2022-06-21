package dotnetcoresdk

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(name string, entries []packit.BuildpackPlanEntry, priorites []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(name string, entries []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface DependencyMapper --output fakes/dependency_mapper.go
type DependencyMapper interface {
	FindCorrespondingVersion(path, versionKey string) (string, error)
}

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Resolve(path, id, version, stack string) (postal.Dependency, error)
	Deliver(dependency postal.Dependency, cnbPath, layerPath, platformPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

//go:generate faux --interface DotnetSymlinker --output fakes/dotnet_symlinker.go
type DotnetSymlinker interface {
	Link(workingDir, layerPath string) error
}

//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go
type SBOMGenerator interface {
	GenerateFromDependency(dependency postal.Dependency, dir string) (sbom.SBOM, error)
}

func Build(entryResolver EntryResolver,
	dependencyMapper DependencyMapper,
	dependencyManager DependencyManager,
	dotnetSymlinker DotnetSymlinker,
	sbomGenerator SBOMGenerator,
	logger scribe.Emitter,
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
				Name: DotnetDependency,
				Metadata: map[string]interface{}{
					"version-source": "RUNTIME_VERSION",
					"version":        sdkVersion,
				},
			})
		}

		planEntry, entries := entryResolver.Resolve(DotnetDependency, context.Plan.Entries, Priorities)
		logger.Candidates(entries)

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
			logger.Subprocess("WARNING: Setting the .NET Core SDK version through buildpack.yml will be deprecated soon in .NET Core SDK Buildpack v%s.", nextMajorVersion.String())
		}

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

		bom := dependencyManager.GenerateBillOfMaterials(sdkDependency)
		launch, build := entryResolver.MergeLayerTypes(DotnetDependency, context.Plan.Entries)

		var buildMetadata packit.BuildMetadata
		if build {
			buildMetadata.BOM = bom
		}

		var launchMetadata packit.LaunchMetadata
		if launch {
			launchMetadata.BOM = bom
		}

		cachedDependencySHA, ok := sdkLayer.Metadata["dependency-sha"]
		if ok && cachedDependencySHA == sdkDependency.SHA256 {
			logger.Process(fmt.Sprintf("Reusing cached layer %s", sdkLayer.Path))
			logger.Break()

			err = dotnetSymlinker.Link(context.WorkingDir, sdkLayer.Path)
			if err != nil {
				return packit.BuildResult{}, err
			}

			envLayer.SharedEnv.Prepend("PATH",
				filepath.Join(context.WorkingDir, ".dotnet_root"),
				string(os.PathListSeparator))

			envLayer.SharedEnv.Override("DOTNET_ROOT", filepath.Join(context.WorkingDir, ".dotnet_root"))
			logger.EnvironmentVariables(envLayer)

			sdkLayer.Build, sdkLayer.Launch, sdkLayer.Cache = build, launch, build || launch

			return packit.BuildResult{
				Layers: []packit.Layer{
					sdkLayer,
					envLayer,
				},
				Build:  buildMetadata,
				Launch: launchMetadata,
			}, nil
		}

		logger.Process("Executing build process")

		sdkLayer, err = sdkLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Subprocess("Installing %s %s", ".NET Core SDK", sdkDependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencyManager.Deliver(sdkDependency, context.CNBPath, sdkLayer.Path, context.Platform.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		sdkLayer.Metadata = map[string]interface{}{
			"dependency-sha": sdkDependency.SHA256,
		}

		err = dotnetSymlinker.Link(context.WorkingDir, sdkLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		sdkLayer.Build, sdkLayer.Launch, sdkLayer.Cache = build, launch, build || launch

		envLayer.SharedEnv.Prepend("PATH",
			filepath.Join(context.WorkingDir, ".dotnet_root"),
			string(os.PathListSeparator))

		envLayer.SharedEnv.Override("DOTNET_ROOT", filepath.Join(context.WorkingDir, ".dotnet_root"))
		logger.EnvironmentVariables(envLayer)

		logger.GeneratingSBOM(sdkLayer.Path)
		var sbomContent sbom.SBOM
		duration, err = clock.Measure(func() error {
			sbomContent, err = sbomGenerator.GenerateFromDependency(sdkDependency, sdkLayer.Path)
			return err
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		logger.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)
		sdkLayer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
		if err != nil {
			return packit.BuildResult{}, err
		}

		return packit.BuildResult{
			Layers: []packit.Layer{
				sdkLayer,
				envLayer,
			},
			Build:  buildMetadata,
			Launch: launchMetadata,
		}, nil
	}
}
