package dotnetcoresdk_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSDKVersionMapper(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer        *bytes.Buffer
		logEmitter    dotnetcoresdk.LogEmitter
		cnbDir        string
		versionMapper dotnetcoresdk.SDKVersionMapper
	)

	it.Before(func() {
		var err error

		buffer = bytes.NewBuffer(nil)
		logEmitter = dotnetcoresdk.NewLogEmitter(buffer)

		versionMapper = dotnetcoresdk.NewSDKVersionMapper(logEmitter)
		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(cnbDir, "buildpack.toml"), []byte(`api = "0.2"
 [buildpack]
   id = "org.some-org.some-buildpack"
   name = "Some Buildpack"
   version = "some-version"

   [[metadata.runtime-to-sdks]]
     runtime-version = "1.2.4"
     sdks = ["1.2.400"]

   [[metadata.runtime-to-sdks]]
     runtime-version = "1.2.3"
     sdks = ["1.2.300"]
 `), 0600)
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	it("returns the SDK version that corresponds to the provided runtime version", func() {
		Expect(cnbDir).NotTo(Equal(""))
		sdkVersion, err := versionMapper.FindCorrespondingVersion(filepath.Join(cnbDir, "buildpack.toml"), "1.2.3")
		Expect(err).NotTo(HaveOccurred())
		Expect(sdkVersion).To(Equal("1.2.300"))
	})

	context("failure cases", func() {

		context("when the buildpack.toml cannot be decoded", func() {
			it.Before(func() {
				err := ioutil.WriteFile(filepath.Join(cnbDir, "buildpack.toml"), []byte(`%%%`), 0600)
				Expect(err).ToNot(HaveOccurred())
			})

			it("returns an error", func() {
				_, err := versionMapper.FindCorrespondingVersion(filepath.Join(cnbDir, "buildpack.toml"), "1.2.3")
				Expect(err).To(MatchError(ContainSubstring("bare keys cannot contain")))
			})
		})

		context("when there is no compatible SDK version in the buildpack.toml", func() {
			it("returns an error", func() {
				_, err := versionMapper.FindCorrespondingVersion(filepath.Join(cnbDir, "buildpack.toml"), "9.9.9")
				Expect(err).To(MatchError(ContainSubstring("no compatible SDK version available for .NET Runtime version 9.9.9")))
			})
		})
	})
}
