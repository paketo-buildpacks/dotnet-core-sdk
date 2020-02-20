package dotnettesting

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
)

func GetLowestRuntimeVersionInMajorMinor(majorMinor, bpTOMLPath string) (string, error) {
	type buildpackTomlVersion struct {
		Metadata struct {
			Dependencies []struct {
				Version string `toml:"version"`
			} `toml:"dependencies"`
		} `toml:"metadata"`
	}

	bpToml := buildpackTomlVersion{}
	output, err := ioutil.ReadFile(filepath.Join(bpTOMLPath))
	if err != nil {
		return "", err
	}

	majorMinorConstraint, err := semver.NewConstraint(fmt.Sprintf("%s.*", majorMinor))
	if err != nil {
		return "", err
	}

	lowestVersion, err := semver.NewVersion(fmt.Sprintf("%s.9999", majorMinor))
	if err != nil {
		return "", err
	}

	_, err = toml.Decode(string(output), &bpToml)
	if err != nil {
		return "", err
	}

	for _, dep := range bpToml.Metadata.Dependencies {
		depVersion, err := semver.NewVersion(dep.Version)
		if err != nil {
			return "", err
		}
		if majorMinorConstraint.Check(depVersion) {
			if depVersion.LessThan(lowestVersion) {
				lowestVersion = depVersion
			}
		}
	}

	return lowestVersion.String(), nil
}

func GetCorrespondingRuntimeFromSDK(sdkVersion, bpTOMLPath string) (string, error) {
	var frameworkVersion string
	var runtimeSDKMap struct {
		Metadata struct {
			RuntimeToSdks []struct {
				RuntimeVersion string   `toml:"runtime-version"`
				Sdks           []string `toml:"sdks"`
			} `toml:"runtime-to-sdks"`
		} `toml:"metadata"`
	}

	_, err := toml.DecodeFile(bpTOMLPath, &runtimeSDKMap)
	if err != nil {
		return "", err
	}

	for _, r := range runtimeSDKMap.Metadata.RuntimeToSdks {
		for _, s := range r.Sdks {
			if s == sdkVersion {
				frameworkVersion = r.RuntimeVersion
			}
		}
	}
	if frameworkVersion == "" {
		return "", fmt.Errorf("no runtime version found for sdk-version %s", sdkVersion)
	}
	return frameworkVersion, nil
}
