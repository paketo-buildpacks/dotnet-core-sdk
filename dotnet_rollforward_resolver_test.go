package dotnetcoresdk_test

import (
	"os"
	"path/filepath"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testRollforwardResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		cnbDir string
	)

	it.Before(func() {
		cnbDir = t.TempDir()

		buildpackPath := filepath.Join(cnbDir, "buildpack.toml")
		err := os.WriteFile(buildpackPath, []byte(`api = "0.2"
			[buildpack]
			id = "org.some-org.some-buildpack"
			name = "Some Buildpack"
			version = "some-version"

			[metadata]
				
			[[metadata.dependencies]]
				id = "dotnet-sdk"
				stacks = ["some-stack"]
				version = "8.0.416"
				
			[[metadata.dependencies]]
				id = "dotnet-sdk"
				stacks = ["some-stack"]
				version = "9.0.307"

			[[metadata.dependencies]]
				id = "dotnet-sdk"
				stacks = ["some-stack"]
				version = "9.0.366"

			[[metadata.dependencies]]
				id = "dotnet-sdk"
				stacks = ["some-stack"]
				version = "9.0.507"

			[[metadata.dependencies]]
				id = "dotnet-sdk"
				stacks = ["some-stack"]
				version = "10.0.100"
		`), 0644)
		Expect(err).NotTo(HaveOccurred())
	})

	context("ResolveWithRollforward", func() {
		it("resolves the correct version based on roll-forward strategy", func() {
			dep, err := dotnetcoresdk.ResolveWithRollforward(
				filepath.Join(cnbDir, "buildpack.toml"),
				"9.0.200",
				"feature",
				"some-stack",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(dep.Version).To(Equal("9.0.366"))
		})

		it("returns an error when no compatible version is found", func() {
			_, err := dotnetcoresdk.ResolveWithRollforward(
				filepath.Join(cnbDir, "buildpack.toml"),
				"8.0.100",
				"patch",
				"some-stack",
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to resolve version 8.0.100 with roll-forward policy 'patch'"))
		})
	})
}
