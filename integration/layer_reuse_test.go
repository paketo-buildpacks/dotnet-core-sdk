package integration_test

import (
	"fmt"
	"io/ioutil"
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
					settings.Buildpacks.DotnetCoreRuntime.Online,
					settings.Buildpacks.DotnetCoreSDK.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(3))
			Expect(firstImage.Buildpacks[1].Key).To(Equal(settings.BuildpackInfo.Buildpack.ID))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("dotnet-core-sdk"))

			// second pack build

			secondImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreRuntime.Online,
					settings.Buildpacks.DotnetCoreSDK.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(3))
			Expect(secondImage.Buildpacks[1].Key).To(Equal(settings.BuildpackInfo.Buildpack.ID))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("dotnet-core-sdk"))

			Expect(logs).To(ContainLines(
				"  Resolving .NET Core SDK version",
				"    Candidate version sources (in priority order):",
				"      <unknown> -> \"*\"",
				"",
				MatchRegexp(`    Selected .NET Core SDK version \(using <unknown>\): \d+\.\d+\.\d+`),
				"",
				MatchRegexp(fmt.Sprintf("  Reusing cached layer /layers/%s/dotnet-core-sdk", strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"))),
				"",
			))
			secondContainer, err = docker.Container.Run.
				WithCommand("ls -al $DOTNET_ROOT && ls -al $DOTNET_ROOT/sdk").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[secondContainer.ID] = struct{}{}

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(secondContainer.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(
				And(
					MatchRegexp(`-rwxr-xr-x \d+ cnb cnb \d+ .* dotnet`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb     \d+ .* packs -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/packs`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb     \d+ .* sdk -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/sdk`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb     \d+ .* templates -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/templates`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb \d+ .* /workspace/.dotnet_root/sdk -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/sdk`),
				),
			)

			Expect(secondImage.Buildpacks[1].Layers["dotnet-core-sdk"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[1].Layers["dotnet-core-sdk"].Metadata["built_at"]))
		})
	})

	context.Focus("when an app is rebuilt with changed requirements", func() {
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

		it.Pend("does not reuse the cached sdk layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			err = ioutil.WriteFile(filepath.Join(source, "buildpack.yml"), []byte(`---
dotnet-framework:
  version: "2.*"
			`), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			firstImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreRuntime.Online,
					settings.Buildpacks.DotnetCoreSDK.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(3))
			Expect(firstImage.Buildpacks[1].Key).To(Equal(settings.BuildpackInfo.Buildpack.ID))
			Expect(firstImage.Buildpacks[1].Layers).To(HaveKey("dotnet-core-sdk"))

			// second pack build
			err = ioutil.WriteFile(filepath.Join(source, "buildpack.yml"), []byte(`---
dotnet-framework:
  version: "5.*"
			`), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			secondImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreRuntime.Online,
					settings.Buildpacks.DotnetCoreSDK.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(3))
			Expect(secondImage.Buildpacks[1].Key).To(Equal(settings.BuildpackInfo.Buildpack.ID))
			Expect(secondImage.Buildpacks[1].Layers).To(HaveKey("dotnet-core-sdk"))

			Expect(logs).To(ContainLines(
				"  Resolving .NET Core SDK version",
				"    Candidate version sources (in priority order):",
				MatchRegexp(`      <unknown> -> "5\.\d+\.\d+", `),
				"",
				MatchRegexp(`    Selected .NET Core SDK version \(using <unknown>\): \d+\.\d+\.\d+`),
			))

			Expect(logs).NotTo(ContainSubstring("Reusing cached layer"))

			secondContainer, err = docker.Container.Run.
				WithCommand("ls -al $DOTNET_ROOT && ls -al $DOTNET_ROOT/sdk").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[secondContainer.ID] = struct{}{}

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(secondContainer.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(
				And(
					MatchRegexp(`-rwxr-xr-x \d+ cnb cnb \d+ .* dotnet`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb     \d+ .* packs -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/packs`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb     \d+ .* sdk -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/sdk`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb     \d+ .* templates -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/templates`),
					MatchRegexp(`lrwxrwxrwx \d+ cnb cnb \d+ .* /workspace/.dotnet_root/sdk -> /layers/paketo-buildpacks_dotnet-core-sdk/dotnet-core-sdk/sdk`),
				),
			)

			Expect(secondImage.Buildpacks[1].Layers["dotnet-core-sdk"].Metadata["built_at"]).NotTo(Equal(firstImage.Buildpacks[1].Layers["dotnet-core-sdk"].Metadata["built_at"]))
		})
	})
}
