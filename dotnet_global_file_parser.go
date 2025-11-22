package dotnetcoresdk

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/Masterminds/semver/v3"
)

type GlobalJson struct {
	Sdk *Sdk `json:"sdk,omitempty"`
}

type Sdk struct {
	Version         *string `json:"version,omitempty"`
	AllowPrerelease *bool   `json:"allowPrerelease,omitempty"`
	RollForward     *string `json:"rollForward,omitempty"`
}

type ConstraintResult struct {
	Constraint string
	Name       string
}

func GetConstraintsFromGlobalJson(global GlobalJson) ([]ConstraintResult, error) {
	results := []ConstraintResult{}

	sdk := global.Sdk
	if sdk == nil {
		return results, nil
	}

	if sdk.Version == nil {
		return results, nil
	}
	version, err := semver.NewVersion(*sdk.Version)
	if err != nil {
		return nil, err
	}

	rollForward := "patch"
	if sdk.RollForward != nil {
		rollForward = *sdk.RollForward
	}

	featureLevel := version.Patch() / 100

	// Refer to the documentation on rollForward behaviour
	// https://learn.microsoft.com/en-us/dotnet/core/tools/global-json#rollforward
	if slices.Contains([]string{"patch", "disabled"}, rollForward) {
		results = append(results, ConstraintResult{
			Constraint: version.String(),
			Name:       "global.json exact",
		})
	}
	if slices.Contains([]string{"patch", "feature", "minor", "major", "latestPatch"}, rollForward) {
		maxPatch := getPatchForFeatureLevel(featureLevel + 1)
		maxVersion := fmt.Sprintf("%d.%d.%d", version.Major(), version.Minor(), maxPatch)

		results = append(results, ConstraintResult{
			Constraint: fmt.Sprintf(">= %s, < %s", version.String(), maxVersion),
			Name:       "global.json patch",
		})
	}
	if slices.Contains([]string{"feature", "minor", "major"}, rollForward) {
		nextFeaturePatch := getPatchForFeatureLevel(featureLevel + 1)
		maxFeaturePatch := getPatchForFeatureLevel(featureLevel + 2)

		minVersion := fmt.Sprintf("%d.%d.%d", version.Major(), version.Minor(), nextFeaturePatch)
		maxVersion := fmt.Sprintf("%d.%d.%d", version.Major(), version.Minor(), maxFeaturePatch)

		results = append(results, ConstraintResult{
			Constraint: fmt.Sprintf(">= %s, < %s", minVersion, maxVersion),
			Name:       "global.json feature",
		})
	}
	if slices.Contains([]string{"minor", "major"}, rollForward) {
		nextMinor := version.Minor() + 1
		minFeaturePatch := getPatchForFeatureLevel(1)
		maxFeaturePatch := getPatchForFeatureLevel(2)

		minVersion := fmt.Sprintf("%d.%d.%d", version.Major(), nextMinor, minFeaturePatch)
		maxVersion := fmt.Sprintf("%d.%d.%d", version.Major(), nextMinor, maxFeaturePatch)

		results = append(results, ConstraintResult{
			Constraint: fmt.Sprintf(">= %s, < %s", minVersion, maxVersion),
			Name:       "global.json minor",
		})
	}
	if rollForward == "major" {
		nextMajor := version.Major() + 1
		minFeaturePatch := getPatchForFeatureLevel(1)
		maxFeaturePatch := getPatchForFeatureLevel(2)

		minVersion := fmt.Sprintf("%d.0.%d", nextMajor, minFeaturePatch)
		maxVersion := fmt.Sprintf("%d.0.%d", nextMajor, maxFeaturePatch)

		results = append(results, ConstraintResult{
			Constraint: fmt.Sprintf(">= %s, < %s", minVersion, maxVersion),
			Name:       "global.json major",
		})
	}
	if rollForward == "latestFeature" {
		maxVersion := fmt.Sprintf("%d.%d.*", version.Major(), version.Minor())

		results = append(results, ConstraintResult{
			Constraint: fmt.Sprintf(">= %s, %s", version.String(), maxVersion),
			Name:       "global.json feature",
		})
	}
	if rollForward == "latestMinor" {
		maxVersion := fmt.Sprintf("%d.*.*", version.Major())

		results = append(results, ConstraintResult{
			Constraint: fmt.Sprintf(">= %s, %s", version.String(), maxVersion),
			Name:       "global.json minor",
		})
	}
	if rollForward == "latestMajor" {
		results = append(results, ConstraintResult{
			Constraint: fmt.Sprintf(">= %s", version.String()),
			Name:       "global.json major",
		})
	}

	return results, nil
}

func FindGlobalJson(dir string) (*GlobalJson, error) {
	filePath := path.Join(dir, "global.json")
	if _, err := os.Stat(filePath); err == nil {
		jsonFile, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load global.json: %w", err)
		}

		fileContents, err := io.ReadAll(jsonFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read global.json: %w", err)
		}

		var globalJson GlobalJson

		err = json.Unmarshal(fileContents, &globalJson)
		if err != nil {
			return nil, fmt.Errorf("failed to parse global.json: %w", err)
		}

		return &globalJson, nil
	}

	parentDir := filepath.Dir(dir)

	if dir == parentDir {
		return nil, nil
	}

	// Recurse up the tree to try find global.json
	return FindGlobalJson(filepath.Dir(dir))
}

func getPatchForFeatureLevel(featureLevel uint64) uint64 {
	return featureLevel * 100
}
