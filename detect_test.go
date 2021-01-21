package dotnetcoresdk_test

import (
	"errors"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/dotnet-core-sdk/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buildpackYMLParser *fakes.BuildpackYMLParser

		detect packit.DetectFunc
	)

	it.Before(func() {
		buildpackYMLParser = &fakes.BuildpackYMLParser{}
		detect = dotnetcoresdk.Detect(buildpackYMLParser)
	})

	it("provides the dotnet-sdk as a dependency and requires dotnet-runtime at build", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: "working-dir",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Plan).To(Equal(packit.BuildPlan{
			Requires: []packit.BuildPlanRequirement{
				{
					Name: "dotnet-runtime",
					Metadata: map[string]interface{}{
						"build": true,
					},
				},
			},
			Provides: []packit.BuildPlanProvision{
				{Name: "dotnet-sdk"},
			},
		}))
	})

	context("when a version is specified in the buildpack.yml", func() {
		it.Before(func() {
			buildpackYMLParser.ParseCall.Returns.String = "some-version"
		})

		it("requires the version of the SDK specified in the buildpack.yml", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: "working-dir",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: "dotnet-sdk"},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "dotnet-runtime",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
					{
						Name: "dotnet-sdk",
						Metadata: map[string]interface{}{
							"version":        "some-version",
							"version-source": "buildpack.yml",
						},
					},
				},
			}))
			Expect(buildpackYMLParser.ParseCall.Receives.WorkingDir).To(Equal("working-dir"))
		})
	})

	context("failure cases", func() {
		context("when buildpackYML can't be parsed", func() {
			it.Before(func() {
				buildpackYMLParser.ParseCall.Returns.Error = errors.New("some-error")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: "working-dir",
				})
				Expect(err).To(MatchError("some-error"))
			})
		})
	})
}
