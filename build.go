package dotnetcoresdk

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
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

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Resolve(path, id, version, stack string) (postal.Dependency, error)
	Deliver(dependency postal.Dependency, cnbPath, layerPath, platformPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go
type SBOMGenerator interface {
	GenerateFromDependency(dependency postal.Dependency, dir string) (sbom.SBOM, error)
}

func Build(entryResolver EntryResolver,
	dependencyManager DependencyManager,
	sbomGenerator SBOMGenerator,
	logger scribe.Emitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)
		logger.Process("Resolving .NET Core SDK version")

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

		dependencyChecksum := sdkDependency.Checksum
		if sdkDependency.SHA256 != "" {
			dependencyChecksum = sdkDependency.SHA256
		}

		cachedChecksum, ok := sdkLayer.Metadata["dependency-checksum"].(string)

		if ok && cargo.Checksum(dependencyChecksum).MatchString(cachedChecksum) {
			logger.Process(fmt.Sprintf("Reusing cached layer %s", sdkLayer.Path))
			logger.Break()

			sdkLayer.Build, sdkLayer.Launch, sdkLayer.Cache = build, launch, build || launch

			return packit.BuildResult{
				Layers: []packit.Layer{
					sdkLayer,
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
			"dependency-checksum": dependencyChecksum,
		}

		sdkLayer.Build, sdkLayer.Launch, sdkLayer.Cache = build, launch, build || launch

		sdkLayer.BuildEnv.Prepend("PATH", sdkLayer.Path, string(os.PathListSeparator))
		logger.EnvironmentVariables(sdkLayer)

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
			},
			Build:  buildMetadata,
			Launch: launchMetadata,
		}, nil
	}
}
