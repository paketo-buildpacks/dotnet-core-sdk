package utils

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/cloudfoundry/libcfbuildpack/build"
)

func BuildpackYAMLVersionCheck(versionRuntimeConfig, versionBuildpackYAML string) error {
	runtimeVersion, err := semver.NewVersion(versionRuntimeConfig)
	if err != nil {
		return err
	}

	buildpackYAMLVersion, err := semver.NewVersion(versionBuildpackYAML)
	if err != nil {
		return err
	}

	if runtimeVersion.Major() != buildpackYAMLVersion.Major(){
		return fmt.Errorf("major versions of runtimes do not match between buildpack.yml and runtimeconfig.json")
	}

	if buildpackYAMLVersion.Minor() < runtimeVersion.Minor() {
		return fmt.Errorf("the minor version of the runtimeconfig.json is greater than the minor version of the buildpack.yml")
	}

	return nil
}

func FrameworkRollForward(version, framework string, context build.Build) (string, error) {
	splitVersion, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}
	anyPatch := fmt.Sprintf("%d.%d.*", splitVersion.Major(), splitVersion.Minor())
	anyMinor := fmt.Sprintf("%d.*.*", splitVersion.Major())

	versions := []string{version, anyPatch, anyMinor}

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return "", err
	}

	for _, versionConstraint := range versions {
		highestVersion, err := deps.Best(framework, versionConstraint, context.Stack)
		if err == nil {
			if highestVersion.Version.Minor() == splitVersion.Minor() && highestVersion.Version.Patch() < splitVersion.Patch(){
				continue
			}
			if highestVersion.Version.Minor() < splitVersion.Minor() {
				break
			}
			return highestVersion.Version.String(), nil
		}
	}

	return "", fmt.Errorf("no compatible versions found")
}
