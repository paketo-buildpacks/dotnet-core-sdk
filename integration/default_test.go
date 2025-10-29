package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	. "github.com/paketo-buildpacks/occam/matchers"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDefault(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually
		pack       occam.Pack
		docker     occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when building a container with dotnet sdk", func() {
		var (
			image     occam.Image
			container occam.Container

			name    string
			source  string
			sbomDir string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			sbomDir, err = os.MkdirTemp("", "sbom")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chmod(sbomDir, os.ModePerm)).To(Succeed())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())

			Expect(os.RemoveAll(source)).To(Succeed())
			Expect(os.RemoveAll(sbomDir)).To(Succeed())
		})

		it("builds an oci image with dotnet-sdk installed", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreSDK.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithSBOMOutputDir(sbomDir).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.BuildpackInfo.Buildpack.Name)),
				"  Resolving .NET Core SDK version",
				"    Candidate version sources (in priority order):",
				"      <unknown> -> \"\"",
				"",
				MatchRegexp(`    Selected .NET Core SDK version \(using <unknown>\): \d+\.\d+\.\d+`),
			))
			Expect(logs).To(ContainLines(
				"  Executing build process",
				MatchRegexp(`    Installing .NET Core SDK \d+\.\d+\.\d+`),
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				"",
				"  Configuring build environment",
				fmt.Sprintf(`    PATH -> "/layers/%s/dotnet-core-sdk:$PATH"`, strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_")),
			))

			container, err = docker.Container.Run.
				WithCommand(fmt.Sprintf(`ls -al /layers/%s/dotnet-core-sdk && ls -al /layers/%s/dotnet-core-sdk/sdk`,
					strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"),
					strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"))).
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(
				And(
					// Note: The assumption here is that the file permissions for the dotnet CLI below (-rwxr-xr-x)
					// and its existence in the .dotnet_root directory (which is on the PATH) sufficiently proves
					// its ability to be called. This may need refactoring if that assumption is proved insufficient.
					MatchRegexp(`-rwxr-xr-x \d+ \w+ cnb\s+\d+ .* dotnet`),
					MatchRegexp(`drwxr-xr-x \d+ \w+ cnb\s+\d+ .* host`),
					MatchRegexp(`drwxr-xr-x \d+ \w+ cnb\s+\d+ .* sdk`),
				),
			)

			contents, err := os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", "sbom.legacy.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(`"name":".NET Core SDK"`))

			// check that all required SBOM files are present
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"), "dotnet-core-sdk", "sbom.cdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"), "dotnet-core-sdk", "sbom.spdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"), "dotnet-core-sdk", "sbom.syft.json")).To(BeARegularFile())

			// check an SBOM file
			contents, err = os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"), "dotnet-core-sdk", "sbom.cdx.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(`"name": ".NET Core SDK"`))
		})
	})
}
