package dotnetcoresdk

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func Detect(logger scribe.Emitter) packit.DetectFunc {
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
		if globalJson != nil && globalJson.Sdk != nil && globalJson.Sdk.Version != nil {
			version, err := semver.NewVersion(*globalJson.Sdk.Version)
			if err != nil {
				return packit.DetectResult{}, err
			}

			rollForward := "patch"
			if globalJson.Sdk.RollForward != nil {
				rollForward = *globalJson.Sdk.RollForward
			}

			plan.Requires = append(plan.Requires, packit.BuildPlanRequirement{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version":        version.String(),
					"version-source": "global.json",
					"roll-forward":   rollForward,
				},
			})
		}

		if sdkVersion, ok := os.LookupEnv(DotnetSdkVersion); ok {
			_, err := semver.NewConstraint(sdkVersion)
			if err != nil {
				return packit.DetectResult{}, err
			}
			plan.Requires = append(plan.Requires, packit.BuildPlanRequirement{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": DotnetSdkVersion,
					"version":        sdkVersion,
				},
			})
		}

		if frameworkVersion, ok := os.LookupEnv(DeprecatedFrameworkVersion); ok {
			logger.Subprocess(scribe.YellowColor("WARNING: BP_DOTNET_FRAMEWORK_VERSION is deprecated and will be removed in a future version. Please use global.json or BP_DOTNET_SDK_VERSION to select SDK version instead."))
			frameworkSemver, err := semver.NewVersion(frameworkVersion)
			if err != nil {
				return packit.DetectResult{}, err
			}
			plan.Requires = append(plan.Requires, packit.BuildPlanRequirement{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": DeprecatedFrameworkVersion,
					"version":        fmt.Sprintf("%d.%d.*", frameworkSemver.Major(), frameworkSemver.Minor()),
				},
			})
		}

		return packit.DetectResult{Plan: plan}, nil
	}
}
