package sdk

import (
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"path/filepath"
	"testing"
)

func TestUnitSdk(t *testing.T) {
	spec.Run(t, "Detect", testSdk, spec.Report(report.Terminal{}))
}

func testSdk(t *testing.T, when spec.G, it spec.S) {
	var (
		factory                 *test.BuildFactory
		stubDotnetSDKFixture      = filepath.Join("testdata", "stub-sdk-dependency.tar.xz")
		fakeSymlinkTarget       string
		runtimeSymlinkLayerPath string
		symlinkLayer            layers.Layer
	)

	it.Before(func() {
		var err error

		RegisterTestingT(t)
		factory = test.NewBuildFactory(t)
		factory.AddDependency(DotnetSDK, stubDotnetSDKFixture)
		symlinkLayer = factory.Build.Layers.Layer("driver-symlinks")


		fakeSymlinkTarget, err = ioutil.TempDir("", "")
		runtimeSymlinkLayerPath, err = ioutil.TempDir(os.TempDir(), "runtime")
		Expect(err).ToNot(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(runtimeSymlinkLayerPath, "shared"), os.ModePerm)).To(Succeed())
		Expect(os.Symlink(fakeSymlinkTarget, filepath.Join(runtimeSymlinkLayerPath, "shared", "Microsoft.NETCore.App"))).To(Succeed())

		os.Setenv("DOTNET_ROOT", runtimeSymlinkLayerPath)
	})

	it.After(func () {
		os.RemoveAll(fakeSymlinkTarget)
		os.RemoveAll(runtimeSymlinkLayerPath)
		os.Unsetenv("DOTNET_ROOT")
	})

	when("runtime.NewContributor", func() {
		it("returns true if a build plan exists", func() {
			factory.AddPlan(buildpackplan.Plan{Name: DotnetSDK})

			_, willContribute, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeTrue())
		})

		it("returns false if a build plan does not exist", func() {
			_, willContribute, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeFalse())
		})
	})

	when("Contribute", func() {
		it("appends dotnet driver to path, installs the runtime dependency", func() {
			factory.AddPlan(buildpackplan.Plan{Name: DotnetSDK})

			dotnetRuntimeContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetRuntimeContributor.Contribute()).To(Succeed())

			ExpectSymlink(filepath.Join(symlinkLayer.Root, "host"),t)

			Expect(filepath.Join(symlinkLayer.Root, "shared")).To(BeADirectory())

			ExpectSymlink(filepath.Join(symlinkLayer.Root, "shared", "Microsoft.NETCore.App"),t)

			Expect(symlinkLayer).To(test.HaveAppendPathSharedEnvironment("PATH", filepath.Join(symlinkLayer.Root)))
			Expect(symlinkLayer).To(test.HaveOverrideSharedEnvironment("DOTNET_ROOT", filepath.Join(symlinkLayer.Root)))
		})

		it("uses the default version when a version is not requested", func() {
			factory.AddDependencyWithVersion(DotnetSDK, "0.9", filepath.Join("testdata", "stub-sdk-dependency.tar.xz"))
			factory.SetDefaultVersion(DotnetSDK, "0.9")
			factory.AddPlan(buildpackplan.Plan{Name: DotnetSDK})

			dotnetRuntimeContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetRuntimeContributor.Contribute()).To(Succeed())
			layer := factory.Build.Layers.Layer(DotnetSDK)
			Expect(layer).To(test.HaveLayerVersion("0.9"))
		})

		it("contributes dotnet runtime to the build layer when included in the build plan", func() {
			factory.AddPlan(buildpackplan.Plan{
				Name: DotnetSDK,
				Metadata: buildpackplan.Metadata{
					"build": true,
				},
			})

			dotnetRuntimeContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetRuntimeContributor.Contribute()).To(Succeed())

			layer := factory.Build.Layers.Layer(DotnetSDK)
			Expect(layer).To(test.HaveLayerMetadata(true, false, false))
		})

		it("contributes dotnet runtime to the launch layer when included in the build plan", func() {
			factory.AddPlan(buildpackplan.Plan{
				Name: DotnetSDK,
				Metadata: buildpackplan.Metadata{
					"launch": true,
				},
			})

			dotnetRuntimeContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetRuntimeContributor.Contribute()).To(Succeed())

			layer := factory.Build.Layers.Layer(DotnetSDK)
			Expect(layer).To(test.HaveLayerMetadata(false, false, true))
		})

		it("returns an error when unsupported version of dotnet runtime is included in the build plan", func() {
			factory.AddPlan(buildpackplan.Plan{
				Name:    DotnetSDK,
				Version: "9000.0.0",
				Metadata: buildpackplan.Metadata{
					"launch": true,
				},
			})

			_, shouldContribute, err := NewContributor(factory.Build)
			Expect(err).To(HaveOccurred())
			Expect(shouldContribute).To(BeFalse())
		})
	})
}

func ExpectSymlink(path string, t *testing.T) {
	t.Helper()
	hostFileInfo, err := os.Stat(path)
	Expect(err).ToNot(HaveOccurred())
	Expect(hostFileInfo.Mode() & os.ModeSymlink).ToNot(Equal(0))
}