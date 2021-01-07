package dotnetcoresdk_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSDKVersionResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer          *bytes.Buffer
		logEmitter      dotnetcoresdk.LogEmitter
		cnbDir          string
		versionResolver dotnetcoresdk.SDKVersionResolver
		entry           packit.BuildpackPlanEntry
	)

	it.Before(func() {
		var err error

		buffer = bytes.NewBuffer(nil)
		logEmitter = dotnetcoresdk.NewLogEmitter(buffer)

		versionResolver = dotnetcoresdk.NewSDKVersionResolver(logEmitter)
		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(cnbDir, "buildpack.toml"), []byte(`api = "0.2"
 [buildpack]
   id = "org.some-org.some-buildpack"
   name = "Some Buildpack"
   version = "some-version"

 [metadata]
   [metadata.default-versions]
 	  dotnet-sdk = "1.*"

   [[metadata.dependencies]]
     id = "dotnet-sdk"
     sha256 = "some-sha"
     stacks = ["some-stack"]
     uri = "some-uri"
     version = "1.2.300"

   [[metadata.dependencies]]
     id = "dotnet-sdk"
     sha256 = "some-sha"
     stacks = ["some-stack"]
     uri = "some-uri"
     version = "1.2.400"

   [[metadata.runtime-to-sdks]]
     runtime-version = "1.2.4"
     sdks = ["1.2.400"]

   [[metadata.runtime-to-sdks]]
     runtime-version = "1.2.3"
     sdks = ["1.2.300"]
 `), 0600)
		Expect(err).NotTo(HaveOccurred())

		entry = packit.BuildpackPlanEntry{
			Name: "dotnet-sdk",
			Metadata: map[string]interface{}{
				"version-source": "global.json",
				"launch":         true,
			},
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	context("when the RUNTIME_VERSION variable is set", func() {
		it.Before(func() {
			Expect(os.Setenv("RUNTIME_VERSION", "1.2.3")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("RUNTIME_VERSION")).To(Succeed())
		})

		context("when the Buildpack Plan entry requires a compatible SDK version", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.300"
			})

			it("returns the compatible SDK version", func() {
				Expect(cnbDir).NotTo(Equal(""))
				dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())
				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-sdk",
					Version: "1.2.300",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})

		context("when the Buildpack Plan entry requires an incompatible SDK version", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.400"
			})

			it("returns an error", func() {
				Expect(cnbDir).NotTo(Equal(""))
				result, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(result).To(Equal(postal.Dependency{}))
				Expect(err).To(MatchError("SDK version specified in global.json (1.2.400) is incompatible with installed runtime version (1.2.3)"))
			})
		})

		context("when the requested SDK version is NOT available", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.500"
			})

			it("returns an error", func() {
				Expect(cnbDir).NotTo(Equal(""))
				result, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(result).To(Equal(postal.Dependency{}))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("no compatible version")))
			})
		})

		context("when no version is requested", func() {
			it("returns the SDK version compatible with the installed Runtime", func() {
				Expect(cnbDir).NotTo(Equal(""))
				dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())
				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-sdk",
					Version: "1.2.300",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})

		context("when the buildpack.toml does not have a dependency with a matching stack", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.3"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "random-stack")
				Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"dotnet-sdk\" dependency for stack \"random-stack\" with version constraint \"1.2.3\": no compatible versions. Supported versions are: []")))
			})
		})

	}, spec.Sequential())

	context("when the RUNTIME_VERSION variable is not set", func() {

		context("when the requested SDK version is available", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.400"
			})

			it("returns the requested SDK version", func() {
				dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())
				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-sdk",
					Version: "1.2.400",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})

		context("when the requested SDK version is NOT available", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.500"
			})

			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("no compatible version")))
			})
		})

		context("when no version is requested", func() {
			it.Before(func() {
				entry.Metadata["version"] = ""
			})

			it("returns the default SDK version", func() {
				dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())
				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-sdk",
					Version: "1.2.400",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})

		context("when the buildpack.toml does not have a dependency with a matching stack", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.3"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "random-stack")
				Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"dotnet-sdk\" dependency for stack \"random-stack\" with version constraint \"1.2.3\": no compatible versions. Supported versions are: []")))
			})
		})

	})

	context("failure cases", func() {

		context("when the buildpack.toml cannot be decoded", func() {

			it.Before(func() {
				err := ioutil.WriteFile(filepath.Join(cnbDir, "buildpack.toml"), []byte(`%%%`), 0600)
				Expect(err).ToNot(HaveOccurred())
			})

			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "random-stack")
				Expect(err).To(MatchError(ContainSubstring("bare keys cannot contain")))
			})

		})

	})
}
