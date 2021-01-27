package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/occam"
	. "github.com/paketo-buildpacks/occam/matchers"
	"github.com/sclevine/spec"
)

func testOffline(t *testing.T, context spec.G, it spec.S) {
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

	context("when building a container with dotnet-sdk in an offline setting", func() {
		var (
			image     occam.Image
			container occam.Container
			name      string
			source    string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("installs the dotnet sdk into a layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreRuntime.Offline,
					settings.Buildpacks.DotnetCoreSDK.Offline,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithNetwork("none").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.BuildpackInfo.Buildpack.Name)),
				"  Resolving .NET Core SDK version",
				"    Candidate version sources (in priority order):",
				MatchRegexp(`      RUNTIME_VERSION -> "\d+\.\d+\.\d+"`),
				"      <unknown>       -> \"*\"",
				"",
				MatchRegexp(`    Selected .NET Core SDK version \(using RUNTIME_VERSION\): \d+\.\d+\.\d+`),
				"",
				"  Executing build process",
				MatchRegexp(`    Installing .NET Core SDK \d+\.\d+\.\d+`),
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				"",
				"  Configuring environment",
				`    DOTNET_ROOT -> "/workspace/.dotnet_root"`,
				`    PATH        -> "/workspace/.dotnet_root:$PATH"`,
			))

			container, err = docker.Container.Run.
				WithCommand("ls -al $DOTNET_ROOT && ls -al $DOTNET_ROOT/sdk").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(
				And(
					MatchRegexp(`-rwxr-xr-x \d+ cnb cnb \d+ .* dotnet`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb \s+\d+ .* packs -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/packs`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb \s+\d+ .* sdk -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/sdk`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb \s+\d+ .* templates -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/templates`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb \d+ .* /workspace/.dotnet_root/sdk -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/sdk`),
				),
			)
		})
	})
}
