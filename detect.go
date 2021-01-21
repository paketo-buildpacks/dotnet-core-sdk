package dotnetcoresdk

import "github.com/paketo-buildpacks/packit"

//go:generate faux --interface BuildpackYMLParser --output fakes/buildpack_yml_parser.go
type BuildpackYMLParser interface {
	Parse(workingDir string) (string, error)
}

func Detect(buildpackYMLParser BuildpackYMLParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.BuildPlan{
			Requires: []packit.BuildPlanRequirement{
				{
					Name: "dotnet-runtime",
					Metadata: map[string]interface{}{
						"build": true,
					},
				},
			},
			Provides: []packit.BuildPlanProvision{
				{
					Name: "dotnet-sdk",
				},
			},
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
