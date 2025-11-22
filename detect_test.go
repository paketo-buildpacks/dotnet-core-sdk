package dotnetcoresdk_test

import (
	"os"
	"path/filepath"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		detect packit.DetectFunc
	)

	it.Before(func() {
		detect = dotnetcoresdk.Detect()
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

	context("when a global.json file is provided", func() {
		it("requires the version specified in the global.json file", func() {
			tempDir := t.TempDir()
			err := os.WriteFile(filepath.Join(tempDir, "global.json"), []byte(`{
				"sdk": {
					"version": "7.0.203",
					"rollForward": "patch"
				}
			}`), 0644)
			Expect(err).NotTo(HaveOccurred())

			result, err := detect(packit.DetectContext{
				WorkingDir: tempDir,
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
							"version":        "7.0.203",
							"version-source": "global.json",
							"roll-forward":   "patch",
						},
					},
				},
			}))
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
	})
}
