package dotnetcoresdk

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
)

type RuntimeToSdks struct {
	RuntimeVersion string   `toml:"runtime-version"`
	SDKs           []string `toml:"sdks"`
}

type SDKVersionResolver struct {
	logger LogEmitter
}

func NewSDKVersionResolver(logger LogEmitter) SDKVersionResolver {
	return SDKVersionResolver{logger: logger}
}

func (r SDKVersionResolver) Resolve(path string,
	entry packit.BuildpackPlanEntry,
	stack string) (postal.Dependency, error) {

	var sdkVersion string
	if sdkVersionStruct, ok := entry.Metadata["version"]; ok {
		sdkVersion = sdkVersionStruct.(string)
	}

	if sdkVersion == "" || sdkVersion == "default" {
		sdkVersion = "*"
	}

	sdkConstraint, err := semver.NewConstraint(sdkVersion)
	if err != nil {
		return postal.Dependency{}, err
	}

	availableDependencies, err := gatherDependenciesFromBuildpackTOML(path, "dotnet-sdk", stack) // get the dependency from the buildpack.toml and return it
	if err != nil {
		return postal.Dependency{}, fmt.Errorf("buildpack.toml could not be parsed: %w", err)
	}

	var sdkDependencies []postal.Dependency
	for _, dep := range availableDependencies {
		depVersion, err := semver.NewVersion(dep.Version)
		if err != nil {
			return postal.Dependency{}, err
		}
		if sdkConstraint.Check(depVersion) {
			sdkDependencies = append(sdkDependencies, dep)
		}
	}

	if len(sdkDependencies) == 0 {
		var supportedVersions []string
		for _, dependency := range availableDependencies {
			supportedVersions = append(supportedVersions, dependency.Version)
		}

		return postal.Dependency{}, fmt.Errorf(
			"failed to satisfy %q dependency for stack %q with version constraint %q: no compatible versions. Supported versions are: [%s]",
			entry.Name,
			stack,
			sdkVersion,
			strings.Join(supportedVersions, ", "),
		)
	}
	if runtimeVersion := os.Getenv("RUNTIME_VERSION"); runtimeVersion != "" {
		var buildpackTOML struct {
			Metadata struct {
				RuntimeToSdks []RuntimeToSdks `toml:"runtime-to-sdks"`
			} `toml:"metadata"`
		}

		_, err := toml.DecodeFile(path, &buildpackTOML)
		if err != nil {
			return postal.Dependency{}, fmt.Errorf("buildpack.toml could not be parsed: %w", err)
		}

		runtimeToSDKVersion := map[string]string{}

		for _, mapping := range buildpackTOML.Metadata.RuntimeToSdks {
			runtimeToSDKVersion[mapping.RuntimeVersion] = mapping.SDKs[0]
		}

		compatibleSDKVersion := runtimeToSDKVersion[runtimeVersion]

		compatibleSDKConstraint, err := semver.NewConstraint(compatibleSDKVersion)
		if err != nil {
			return postal.Dependency{}, err
		}

		for _, dep := range sdkDependencies {
			depVersion, err := semver.NewVersion(dep.Version)
			if err != nil {
				return postal.Dependency{}, err
			}
			if compatibleSDKConstraint.Check(depVersion) {
				return dep, nil
			}
		}

		var sdkVersionSource string
		if sdkVersionSourceStruct, ok := entry.Metadata["version-source"]; ok {
			sdkVersionSource = sdkVersionSourceStruct.(string)
		}

		return postal.Dependency{}, fmt.Errorf("SDK version specified in %s (%s) is incompatible with installed runtime version (%s)",
			sdkVersionSource, sdkVersion, runtimeVersion)
	}
	sort.Slice(sdkDependencies, func(i, j int) bool {
		iVersion := semver.MustParse(sdkDependencies[i].Version)
		jVersion := semver.MustParse(sdkDependencies[j].Version)
		return iVersion.GreaterThan(jVersion)
	})

	return sdkDependencies[0], nil
}

func containsStack(stacks []string, stack string) bool {
	for _, s := range stacks {
		if s == stack {
			return true
		}
	}
	return false
}

func gatherDependenciesFromBuildpackTOML(path, dependencyID, stack string) ([]postal.Dependency, error) {
	var buildpackTOML struct {
		Metadata struct {
			Dependencies []postal.Dependency `toml:"dependencies"`
		} `toml:"metadata"`
	}

	_, err := toml.DecodeFile(path, &buildpackTOML)
	if err != nil {
		return []postal.Dependency{}, err
	}

	var filteredDependencies []postal.Dependency
	for _, dependency := range buildpackTOML.Metadata.Dependencies {
		if dependency.ID == dependencyID && containsStack(dependency.Stacks, stack) {
			filteredDependencies = append(filteredDependencies, dependency)
		}
	}
	return filteredDependencies, nil
}
