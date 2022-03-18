package dotnetcoresdk_test

import (
	"os"
	"path/filepath"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSdkVersionParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string

		sdkVersionParser dotnetcoresdk.SdkVersionParser
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`---
dotnet-sdk:
  version: some-version
`), 0600)).To(Succeed())

		sdkVersionParser = dotnetcoresdk.NewSdkVersionParser()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("finds the buildpack.yml and parses a sdk version from it", func() {
		version, err := sdkVersionParser.Parse(workingDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(version).To(Equal("some-version"))
	})

	context("when there is no buildpack.yml", func() {
		it.Before(func() {
			os.RemoveAll(filepath.Join(workingDir, "buildpack.yml"))
		})
		it("returns an empty string and no error", func() {
			version, err := sdkVersionParser.Parse(workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(""))
		})
	})

	context("failure cases", func() {
		context("when the buildpack.yml is malformed", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "buildpack.yml"), []byte(`[[[`), 0600)).To(Succeed())
			})

			it("returns the error", func() {
				_, err := sdkVersionParser.Parse(workingDir)
				Expect(err).To(MatchError(ContainSubstring("did not find expected node content")))
			})
		})

		context("when the buildpack.yml cannot be opened", func() {
			it.Before(func() {
				Expect(os.Chmod(filepath.Join(workingDir, "buildpack.yml"), 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(workingDir, "buildpack.yml"), 0600)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := sdkVersionParser.Parse(workingDir)
				Expect(err).To(MatchError(ContainSubstring("could not open buildpack.yml:")))
			})
		})
	})
}
