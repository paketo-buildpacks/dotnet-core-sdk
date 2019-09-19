package sdk

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"regexp"
	"strings"
)

const (
	IncompatibleGlobalAndBuildpackYml = "the versions specfied in global.json and buildpack.yml are incompatible, please reconfigure"
)

func GetLatestCompatibleSDKConstraint(sdkVersion string) (string, error) {
	splitVersion, err := semver.NewVersion(sdkVersion)
	if err != nil {
		return "", err
	}

	compatibleVersionConstraint := fmt.Sprintf("%d.%d.*", splitVersion.Major(), splitVersion.Minor())

	return compatibleVersionConstraint, nil
}

func IsCompatibleSDKOptionWithRuntime(constraintVersion, sdkVersion string) (bool, error){
	//Will make sure that version constraint provided by user is compatible with app constraint
	//(i.e. the version provided in buildpack.yml or global.json is the same major.minor as the
	//csproj/runtimeconfig major minor)

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

func GetConstrainedCompatibleSDK(sdkVersion string, context build.Build) (string, error) {
	highestCompatibleVersion, err := semver.NewVersion("0.0.0")
	if err != nil{
		return "", err
	}

	sdkRegex := makeRegex(sdkVersion)

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return "", err
	}

	for _, dep := range deps {
		if sdkRegex.MatchString(dep.Version.Version.String()){
			if dep.Version.Version.GreaterThan(highestCompatibleVersion){
				highestCompatibleVersion = dep.Version.Version
			}
		}
	}

	if highestCompatibleVersion.String() == "0.0.0"{
		return "", fmt.Errorf("no sdk version matching %s found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version", sdkVersion)
	}

	return highestCompatibleVersion.String(), nil
}

func GetFeatureLineConstraint(version string) (string, error){
	sdkVersion, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}

	featureLine := sdkVersion.Patch() / 100

	featureLineConstraint := fmt.Sprintf("%d.%d.%d*", sdkVersion.Major(), sdkVersion.Minor(), featureLine)

	return featureLineConstraint, nil
}

func GetConstrainedCompatibleSDKForGlobalJson(sdkVersion string, context build.Build) (string, error) {
	highestCompatibleVersion, err := semver.NewVersion("0.0.0")
	if err != nil{
		return "", err
	}

	featureLineSdk, err := GetFeatureLineConstraint(sdkVersion)
	if err != nil {
		return "", err
	}

	sdkVersionCheck, err := semver.NewVersion(sdkVersion)
	if err != nil {
		return "", err
	}

	sdkRegex := makeRegex(featureLineSdk)

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return "", err
	}

	for _, dep := range deps {
		if dep.Version.Version.Equal(sdkVersionCheck){
			return sdkVersionCheck.String(), nil
		}
		if sdkRegex.MatchString(dep.Version.Version.String()){
			if dep.Version.Version.GreaterThan(highestCompatibleVersion){
				highestCompatibleVersion = dep.Version.Version
			}
		}
	}

	if highestCompatibleVersion.String() == "0.0.0"{
		return "", fmt.Errorf("no sdk version matching %s found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version", sdkVersion)
	}

	return highestCompatibleVersion.String(), nil
}

func SelectRollStrategy(buildpackYAMLVersion, globalJSONVersion string) (bool, bool, error){
	if !strings.Contains(buildpackYAMLVersion, "*"){
		bpYMLVersion, err := semver.NewVersion(buildpackYAMLVersion)
		if err != nil {
			return false, false, err
		}

		glbJSONVersion, err := semver.NewVersion(globalJSONVersion)
		if err != nil {
			return false, false, err
		}

		if bpYMLVersion.Major() != glbJSONVersion.Major(){
			return false, false, fmt.Errorf(IncompatibleGlobalAndBuildpackYml)
		}

		if bpYMLVersion.Minor() != glbJSONVersion.Minor(){
			return false, false, fmt.Errorf(IncompatibleGlobalAndBuildpackYml)
		}

		if bpYMLVersion.Patch() < glbJSONVersion.Patch(){
			return false, false, fmt.Errorf(IncompatibleGlobalAndBuildpackYml)
		}

		return true, false, nil
	}

	bpYmlRegex := makeRegex(buildpackYAMLVersion)

	if bpYmlRegex.MatchString(globalJSONVersion){
		return false, true, nil
	}
	return false, false, fmt.Errorf(IncompatibleGlobalAndBuildpackYml)
}

func makeRegex(version string) *regexp.Regexp{
	version = strings.ReplaceAll(version, ".", `\.`)
	version = strings.ReplaceAll(version, "*", `.*`)
	return regexp.MustCompile(version)
}
