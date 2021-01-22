package dotnetcoresdk_test

import (
	"bytes"
	"testing"

	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
)

func testPlanEntryResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer   *bytes.Buffer
		resolver dotnetcoresdk.PlanEntryResolver
	)

	it.Before(func() {
		buffer = bytes.NewBuffer(nil)
		resolver = dotnetcoresdk.NewPlanEntryResolver(dotnetcoresdk.NewLogEmitter(buffer))
	})

	context("when buildpack.yml and RUNTIME_VERSION entries are included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "RUNTIME_VERSION",
						"version":        "runtime-version",
					},
				},
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": "RUNTIME_VERSION",
					"version":        "runtime-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      RUNTIME_VERSION -> \"runtime-version\""))
			Expect(buffer.String()).To(ContainSubstring("      buildpack.yml   -> \"buildpack-yml-version\""))
			Expect(buffer.String()).To(ContainSubstring("      <unknown>       -> \"other-version\""))
		})
	})

	context("when a buildpack.yml entry is included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "buildpack-yml-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      buildpack.yml -> \"buildpack-yml-version\""))
			Expect(buffer.String()).To(ContainSubstring("      <unknown>     -> \"other-version\""))
		})
	})

	context("when a buildpack.yml and *sproj are both included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
					},
				},
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "*sproj",
						"version":        "*sproj-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "buildpack-yml-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      buildpack.yml -> \"buildpack-yml-version\""))
			Expect(buffer.String()).To(ContainSubstring("      *sproj        -> \"*sproj-version\""))
			Expect(buffer.String()).To(ContainSubstring("      <unknown>     -> \"other-version\""))
		})
	})

	context("when a buildpack.yml and global.json are both included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
					},
				},
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "global.json",
						"version":        "globaljson-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "buildpack-yml-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      buildpack.yml -> \"buildpack-yml-version\""))
			Expect(buffer.String()).To(ContainSubstring("      global.json   -> \"globaljson-version\""))
			Expect(buffer.String()).To(ContainSubstring("      <unknown>     -> \"other-version\""))
		})
	})
	context("when a global.json and a *sproj are both included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "*sproj",
						"version":        "*sproj-version",
					},
				},
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "global.json",
						"version":        "globaljson-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": "global.json",
					"version":        "globaljson-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      global.json -> \"globaljson-version\""))
			Expect(buffer.String()).To(ContainSubstring("      *sproj      -> \"*sproj-version\""))
			Expect(buffer.String()).To(ContainSubstring("      <unknown>   -> \"other-version\""))
		})
	})

	context("when entry flags differ", func() {
		context("OR's them together on best plan entry", func() {
			it("has all flags", func() {
				entry := resolver.Resolve([]packit.BuildpackPlanEntry{
					{
						Name: "dotnet-sdk",
						Metadata: map[string]interface{}{
							"version-source": "buildpack.yml",
							"version":        "buildpack-yml-version",
						},
					},
					{
						Name: "dotnet-sdk",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
				})
				Expect(entry).To(Equal(packit.BuildpackPlanEntry{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
						"build":          true,
					},
				}))
			})
		})
	})

	context("when an unknown source entry is included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version": "other-version",
				},
			}))
		})
	})
}
