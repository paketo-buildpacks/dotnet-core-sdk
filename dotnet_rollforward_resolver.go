package dotnetcoresdk

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2/postal"
)

func ResolveWithRollforward(path string, version string, rollForward string, stack string) (postal.Dependency, error) {
	sdkDependencies, supportedVersions, err := filterBuildpackTOML(path, DotnetDependency, stack)
	if err != nil {
		return postal.Dependency{}, err
	}

	constraints, err := GetRollforwardConstraints(version, rollForward)
	if err != nil {
		return postal.Dependency{}, err
	}

	// Iterate through each rollforward contstraint to find compatible dependencies
	// The first constraint to match is used even if later constraints would match a newer version
	compatibleVersions := []postal.Dependency{}
	for _, constraint := range constraints {
		for _, dependency := range sdkDependencies {
			depVersion := semver.MustParse(dependency.Version)
			constraintVersion, err := semver.NewConstraint(constraint)
			if err != nil {
				return postal.Dependency{}, err
			}

			if constraintVersion.Check(depVersion) {
				compatibleVersions = append(compatibleVersions, dependency)
			}
		}

		// Stop once a constraint has matched at least one dependency
		if len(compatibleVersions) > 0 {
			break
		}
	}

	if len(compatibleVersions) == 0 {
		return postal.Dependency{}, fmt.Errorf("failed to resolve version %s with roll-forward policy '%s'. Supported versions are: [%s]",
			version,
			rollForward,
			strings.Join(supportedVersions, ", "),
		)
	}

	// return the highest compatible version
	sort.Slice(compatibleVersions, func(i, j int) bool {
		iVersion := semver.MustParse(compatibleVersions[i].Version)
		jVersion := semver.MustParse(compatibleVersions[j].Version)
		return iVersion.GreaterThan(jVersion)
	})

	return compatibleVersions[0], nil

}

func filterBuildpackTOML(path, dependencyID, stack string) ([]postal.Dependency, []string, error) {
	var buildpackTOML struct {
		Metadata struct {
			Dependencies []postal.Dependency `toml:"dependencies"`
		} `toml:"metadata"`
	}

	_, err := toml.DecodeFile(path, &buildpackTOML)
	if err != nil {
		return []postal.Dependency{}, []string{}, err
	}

	var targetOs string
	targetOs = os.Getenv("CNB_TARGET_OS")
	if targetOs == "" {
		targetOs = runtime.GOOS
	}

	var targetArch string
	targetArch = os.Getenv("CNB_TARGET_ARCH")
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}

	var filteredDependencies []postal.Dependency
	var supportedVersions []string
	for _, dependency := range buildpackTOML.Metadata.Dependencies {
		if dependency.ID != dependencyID || !stacksInclude(dependency.Stacks, stack) || !supportsPlatform(targetOs, targetArch, dependency) {
			continue
		}

		filteredDependencies = append(filteredDependencies, dependency)
		supportedVersions = append(supportedVersions, dependency.Version)
	}

	return filteredDependencies, supportedVersions, nil
}

func stacksInclude(stacks []string, stack string) bool {
	for _, s := range stacks {
		if s == stack || s == "*" {
			return true
		}
	}
	return false
}

func supportsPlatform(targetOs, targetArch string, dependency postal.Dependency) bool {

	// Avoid strict checking in case of dependency does not specify OS/Arch
	if dependency.OS == "" && dependency.Arch == "" {
		return true
	}

	if targetOs != dependency.OS || targetArch != dependency.Arch {
		return false
	}

	return true
}
