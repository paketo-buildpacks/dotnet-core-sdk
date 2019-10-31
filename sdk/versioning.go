package sdk

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/cloudfoundry/libcfbuildpack/build"
)

const (
	IncompatibleGlobalAndBuildpackYml = "the versions specfied in global.json and buildpack.yml are incompatible, please reconfigure"
)

func GetSDKFloatVersion(version string) (string, error) {
	splitVersion, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d.%d.*", splitVersion.Major(), splitVersion.Minor()), nil
}

func GetLatestCompatibleSDKDeps(sdkVersion string, context build.Build) ([]*semver.Version, error) {
	compatibleDeps := []*semver.Version{}

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return compatibleDeps, err
	}

	compatibleVersionConstraint, err := semver.NewConstraint(sdkVersion)
	if err != nil {
		return compatibleDeps, err
	}

	for _, dep := range deps {
		if compatibleVersionConstraint.Check(dep.Version.Version) {
			compatibleDeps = append(compatibleDeps, dep.Version.Version)
		}
	}

	if len(compatibleDeps) == 0 {
		return compatibleDeps, fmt.Errorf("no compatible sdk versions found")
	}

	return compatibleDeps, nil
}

//Will make sure that version constraint provided by user is compatible with app constraint
//(i.e. the version provided in buildpack.yml or global.json is the same major.minor as the
//csproj/runtimeconfig major minor)
func IsCompatibleSDKOptionWithRuntime(constraintVersion, sdkVersion string) (bool, error) {
	sdkVersion = strings.ReplaceAll(sdkVersion, "*", "0")

	versionConstraint, err := semver.NewConstraint(constraintVersion)
	if err != nil {
		return false, err
	}

	sdkCheckVersion, err := semver.NewVersion(sdkVersion)
	if err != nil {
		return false, err
	}

	return versionConstraint.Check(sdkCheckVersion), nil
}

func GetConstrainedCompatibleSDK(sdkVersion string, runtimetoSDK map[string][]string, compatibleDeps []*semver.Version) (string, error) {
	highestCompatibleVersion, err := semver.NewVersion("0.0.0")
	if err != nil {
		return "", err
	}

	sdkConstraint := makeRegex(sdkVersion)

	runtimeVersion, varSet := os.LookupEnv("RUNTIME_VERSION")

	listOfSdks := []string{}
	if varSet {
		var found bool
		listOfSdks, found = runtimetoSDK[runtimeVersion]
		if !found {
			return "", fmt.Errorf("no sdk information for the installed runtime found")
		}
	}

	for _, dep := range compatibleDeps {
		// This if statement will be effectively bypassed if the RUNTIME_VERSION var is not set
		if contains(dep.String(), listOfSdks) || !varSet {
			if sdkConstraint.MatchString(dep.String()) {
				if dep.GreaterThan(highestCompatibleVersion) {
					highestCompatibleVersion = dep
				}
			}
		}
	}

	if highestCompatibleVersion.String() == "0.0.0" {
		return "", fmt.Errorf("no sdk version matching %s found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version", sdkVersion)
	}

	return highestCompatibleVersion.String(), nil
}

func getFeatureLineConstraint(version string) (*semver.Constraints, error) {
	sdkVersion, err := semver.NewVersion(version)
	if err != nil {
		return nil, err
	}

	featureLine := sdkVersion.Patch() / 100

	majorMinor := fmt.Sprintf("%d.%d", sdkVersion.Major(), sdkVersion.Minor())

	featureLineConstraint, err := semver.NewConstraint(fmt.Sprintf(">= %s.%d, < %s.%d", majorMinor, featureLine*100, majorMinor, (featureLine+1)*100))

	if err != nil {
		return nil, err
	}

	return featureLineConstraint, nil
}

// This function gets the latest sdk that in the in the same feature line as the version specified in
// global.json. The feature line is indicated by the hundreds place of the sdk path for example
// in sdk 2.2.805 the feture line in 8
// This is how global.json is supposed to roll forward according to Dotnet
func GetConstrainedCompatibleSDKForGlobalJson(sdkVersion string, compatibleDeps []*semver.Version) (string, error) {
	highestCompatibleVersion, err := semver.NewVersion("0.0.0")
	if err != nil {
		return "", err
	}

	featureLineSdk, err := getFeatureLineConstraint(sdkVersion)
	if err != nil {
		return "", err
	}

	sdkVersionCheck, err := semver.NewVersion(sdkVersion)
	if err != nil {
		return "", err
	}

	for _, dep := range compatibleDeps {
		if dep.Equal(sdkVersionCheck) {
			return sdkVersionCheck.String(), nil
		}
		if featureLineSdk.Check(dep) {
			if dep.GreaterThan(highestCompatibleVersion) {
				highestCompatibleVersion = dep
			}
		}
	}

	if highestCompatibleVersion.String() == "0.0.0" || sdkVersionCheck.GreaterThan(highestCompatibleVersion) {
		return "", fmt.Errorf("no sdk version matching %s found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version", sdkVersion)
	}

	return highestCompatibleVersion.String(), nil
}

func SelectRollStrategy(buildpackYAMLVersion, globalJSONVersion string) (bool, bool, error) {
	if !strings.Contains(buildpackYAMLVersion, "*") {
		bpYMLVersion, err := semver.NewVersion(buildpackYAMLVersion)
		if err != nil {
			return false, false, err
		}

		glbJSONVersion, err := semver.NewVersion(globalJSONVersion)
		if err != nil {
			return false, false, err
		}

		if bpYMLVersion.Major() != glbJSONVersion.Major() {
			return false, false, fmt.Errorf(IncompatibleGlobalAndBuildpackYml)
		}

		if bpYMLVersion.Minor() != glbJSONVersion.Minor() {
			return false, false, fmt.Errorf(IncompatibleGlobalAndBuildpackYml)
		}

		if bpYMLVersion.Patch() < glbJSONVersion.Patch() {
			return false, false, fmt.Errorf(IncompatibleGlobalAndBuildpackYml)
		}

		return true, false, nil
	}

	bpYmlRegex := makeRegex(buildpackYAMLVersion)

	if bpYmlRegex.MatchString(globalJSONVersion) {
		return false, true, nil
	}
	return false, false, fmt.Errorf(IncompatibleGlobalAndBuildpackYml)
}

func makeRegex(version string) *regexp.Regexp {
	version = strings.ReplaceAll(version, ".", `\.`)
	version = strings.ReplaceAll(version, "*", `.*`)
	return regexp.MustCompile(version)
}

func contains(searchVersion string, versions []string) bool {
	for _, version := range versions {
		if version == searchVersion {
			return true
		}
	}
	return false
}
