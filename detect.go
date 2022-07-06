package dotnetcoresdk

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
)

//go:generate faux --interface BuildpackYMLParser --output fakes/buildpack_yml_parser.go
type BuildpackYMLParser interface {
	Parse(workingDir string) (string, error)
}

func Detect(buildpackYMLParser BuildpackYMLParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{
					Name: "dotnet-sdk",
				},
			},
		}

		// check if BP_DOTNET_FRAMEWORK_VERSION is set and use the major.minor for the version

		if frameworkVersion, ok := os.LookupEnv("BP_DOTNET_FRAMEWORK_VERSION"); ok {
			frameworkSemver, err := semver.NewVersion(frameworkVersion)
			if err != nil {
				return packit.DetectResult{}, err
			}
			plan.Requires = append(plan.Requires, packit.BuildPlanRequirement{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
					"version":        fmt.Sprintf("%d.%d.*", frameworkSemver.Major(), frameworkSemver.Minor()),
				},
			})
		}

		version, err := buildpackYMLParser.Parse(context.WorkingDir)
		if err != nil {
			return packit.DetectResult{}, err
		}

		if version != "" {
			plan.Requires = append(plan.Requires, packit.BuildPlanRequirement{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version":        version,
					"version-source": "buildpack.yml",
				},
			})
		}

		return packit.DetectResult{Plan: plan}, nil
	}
}
