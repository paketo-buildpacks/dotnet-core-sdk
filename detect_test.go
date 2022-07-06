package dotnetcoresdk_test

import (
	"errors"
	"os"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/dotnet-core-sdk/fakes"
	"github.com/paketo-buildpacks/packit/v2"
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

	it("provides the dotnet-sdk as a dependency and requires nothing", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: "working-dir",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Plan).To(Equal(packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{Name: "dotnet-sdk"},
			},
		}))
	})

	context("when a version is specified via BP_DOTNET_FRAMEWORK_VERSION", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_DOTNET_FRAMEWORK_VERSION", "1.2.3")).To(Succeed())
		})
		it.After(func() {
			Expect(os.Unsetenv("BP_DOTNET_FRAMEWORK_VERSION")).To(Succeed())
		})

		it("requires the major.minor.* version of the SDK specified in the variable", func() {
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
						Name: "dotnet-sdk",
						Metadata: map[string]interface{}{
							"version":        "1.2.*",
							"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
						},
					},
				},
			}))
		})
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
		context("BP_DOTNET_FRAMEWORK_VERSION is not a semantic version", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_DOTNET_FRAMEWORK_VERSION", "bad-version")).To(Succeed())
			})
			it.After(func() {
				Expect(os.Unsetenv("BP_DOTNET_FRAMEWORK_VERSION")).To(Succeed())
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: "working-dir",
				})
				Expect(err).To(MatchError("Invalid Semantic Version"))
			})
		})
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
