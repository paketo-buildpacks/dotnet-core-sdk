package dotnetcoresdk

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
)

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{
					Name: "dotnet-sdk",
				},
			},
		}

		globalJson, err := FindGlobalJson(context.WorkingDir)
		if err != nil {
			return packit.DetectResult{}, err
		}
		if globalJson != nil {
			constraints, err := GetConstraintsFromGlobalJson(*globalJson)
			if err != nil {
				return packit.DetectResult{}, err
			}

			for _, constraint := range constraints {
				plan.Requires = append(plan.Requires, packit.BuildPlanRequirement{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version":        constraint.Constraint,
						"version-source": constraint.Name,
					},
				})
			}
		}

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

		return packit.DetectResult{Plan: plan}, nil
	}
}
