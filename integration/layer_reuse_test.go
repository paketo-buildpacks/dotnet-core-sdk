package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testLayerReuse(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect       = NewWithT(t).Expect
		Eventually   = NewWithT(t).Eventually
		pack         occam.Pack
		docker       occam.Docker
		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
		imageIDs = map[string]struct{}{}
		containerIDs = map[string]struct{}{}
	})

	context("when an app is rebuilt with no changes", func() {
		var (
			firstImage      occam.Image
			secondImage     occam.Image
			secondContainer occam.Container
			name            string
			source          string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			for containerID := range containerIDs {
				Expect(docker.Container.Remove.Execute(containerID)).To(Succeed())
			}

			for imageID := range imageIDs {
				Expect(docker.Image.Remove.Execute(imageID)).To(Succeed())
			}

			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("restores the entire sdk layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			firstImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreSDK.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(2))
			Expect(firstImage.Buildpacks[0].Key).To(Equal(settings.BuildpackInfo.Buildpack.ID))
			Expect(firstImage.Buildpacks[0].Layers).To(HaveKey("dotnet-core-sdk"))

			// second pack build

			secondImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreSDK.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(2))
			Expect(secondImage.Buildpacks[0].Key).To(Equal(settings.BuildpackInfo.Buildpack.ID))
			Expect(secondImage.Buildpacks[0].Layers).To(HaveKey("dotnet-core-sdk"))

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.BuildpackInfo.Buildpack.Name)),
				"  Resolving .NET Core SDK version",
				"    Candidate version sources (in priority order):",
				"      <unknown> -> \"\"",
				"",
				MatchRegexp(`    Selected .NET Core SDK version \(using <unknown>\): 6\.0\.\d+`),
				"",
				MatchRegexp(fmt.Sprintf("  Reusing cached layer /layers/%s/dotnet-core-sdk", strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"))),
				"",
				"  Configuring build environment",
				fmt.Sprintf(`    PATH -> "/layers/%s/dotnet-core-sdk:$PATH"`, strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_")),
				"",
				"  Configuring launch environment",
				`    PATH -> "/workspace/.dotnet_root:$PATH"`,
			))

			secondContainer, err = docker.Container.Run.
				WithCommand(fmt.Sprintf(`ls -al /layers/%s/dotnet-core-sdk`, strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"))).
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[secondContainer.ID] = struct{}{}

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(secondContainer.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(
				And(
					MatchRegexp(`-rwxr-xr-x \d+ \w+ cnb \d+ .* dotnet`),
					MatchRegexp(`drwxr-xr-x \d+ \w+ cnb   \d+ .* sdk`),
				),
			)

			Expect(secondImage.Buildpacks[1].Layers["dotnet-core-sdk"].SHA).To(Equal(firstImage.Buildpacks[1].Layers["dotnet-core-sdk"].SHA))
		})
	})

	// this test is not currently applicable because there's only
	// one version of the .NET SDK that works with jammy
	if !strings.Contains(builder.Local.Stack.ID, "jammy") {
		context("when an app is rebuilt with changed requirements", func() {
			var (
				firstImage      occam.Image
				secondImage     occam.Image
				secondContainer occam.Container
				name            string
				source          string
			)

			it.Before(func() {
				var err error
				name, err = occam.RandomName()
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				for containerID := range containerIDs {
					Expect(docker.Container.Remove.Execute(containerID)).To(Succeed())
				}

				for imageID := range imageIDs {
					Expect(docker.Image.Remove.Execute(imageID)).To(Succeed())
				}

				Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("does not reuse the cached sdk layer", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "default"))
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer

				firstImage, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.DotnetCoreSDK.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithEnv(map[string]string{
						"BP_DOTNET_FRAMEWORK_VERSION": "3.1.2",
					}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String())

				imageIDs[firstImage.ID] = struct{}{}

				Expect(firstImage.Buildpacks).To(HaveLen(2))
				Expect(firstImage.Buildpacks[0].Key).To(Equal(settings.BuildpackInfo.Buildpack.ID))
				Expect(firstImage.Buildpacks[0].Layers).To(HaveKey("dotnet-core-sdk"))

				// second pack build
				secondImage, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.DotnetCoreSDK.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithEnv(map[string]string{
						"BP_DOTNET_FRAMEWORK_VERSION": "6.0.10",
					}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String())

				imageIDs[secondImage.ID] = struct{}{}

				Expect(secondImage.Buildpacks).To(HaveLen(2))
				Expect(secondImage.Buildpacks[0].Key).To(Equal(settings.BuildpackInfo.Buildpack.ID))
				Expect(secondImage.Buildpacks[0].Layers).To(HaveKey("dotnet-core-sdk"))

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.BuildpackInfo.Buildpack.Name)),
					"  Resolving .NET Core SDK version",
					"    Candidate version sources (in priority order):",
					MatchRegexp(`      BP_DOTNET_FRAMEWORK_VERSION -> "6\.0\.\*"`),
					"      <unknown>                   -> \"\"",
					"",
					MatchRegexp(`    Selected .NET Core SDK version \(using BP_DOTNET_FRAMEWORK_VERSION\): 6\.0\.\d+`),
					"",
					"  Executing build process",
				))

				Expect(logs).NotTo(ContainSubstring("Reusing cached layer"))

				secondContainer, err = docker.Container.Run.
					WithCommand(fmt.Sprintf(`ls -al /layers/%s/dotnet-core-sdk && ls -al /layers/%s/dotnet-core-sdk/sdk`,
						strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"),
						strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"))).
					Execute(secondImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[secondContainer.ID] = struct{}{}

				Eventually(func() string {
					cLogs, err := docker.Container.Logs.Execute(secondContainer.ID)
					Expect(err).NotTo(HaveOccurred())
					return cLogs.String()
				}).Should(
					And(
						MatchRegexp(`-rwxr-xr-x \d+ \w+ cnb \d+ .* dotnet`),
						MatchRegexp(`drwxr-xr-x \d+ \w+ cnb   \d+ .* host`),
						MatchRegexp(`drwxr-xr-x \d+ \w+ cnb   \d+ .* packs`),
						MatchRegexp(`drwxr-xr-x \d+ \w+ cnb   \d+ .* sdk`),
						MatchRegexp(`drwxr-xr-x \d+ \w+ cnb   \d+ .* sdk-manifests`),
						MatchRegexp(`drwxr-xr-x \d+ \w+ cnb   \d+ .* shared`),
						MatchRegexp(`drwxr-xr-x \d+ \w+ cnb   \d+ .* templates`),
						MatchRegexp(`drwxr-xr-x \d+ \w+ cnb \d+ .* 6\.0\.\d+`),
					),
				)

				Expect(secondImage.Buildpacks[0].Layers["dotnet-core-sdk"].SHA).NotTo(Equal(firstImage.Buildpacks[0].Layers["dotnet-core-sdk"].SHA))
			})
		})

	}
}
