package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/paketo-buildpacks/dotnet-core-sdk/sdk"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
		fakeBuildpackToml := `
[[dependencies]]
id = "dotnet-sdk"
name = "Dotnet SDK"
stacks = ["org.testeroni"]
uri = "some-uri"
version = "2.2.806"
`
		_, err := toml.Decode(fakeBuildpackToml, &factory.Detect.Buildpack.Metadata)
		Expect(err).ToNot(HaveOccurred())
		factory.Detect.Stack = "org.testeroni"
	})

	when("when there is a valid runtimeconfig.json", func() {
		it("passes when we can find a compatible sdk for the given runtime and there is an executable that with the same name as the app", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
  "runtimeOptions": {
    "tfm": "netcoreapp2.2",
    "framework": {
      "name": "Microsoft.NETCore.App",
      "version": "2.2.5"
    }
  }
}
`), os.ModePerm)).To(Succeed())
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: sdk.DotnetSDK}},
				Requires: []buildplan.Required{{
					Name:     sdk.DotnetSDK,
					Version:  "2.2.5",
					Metadata: buildplan.Metadata{"build": true, "launch": true},
				}, {
					Name:     "dotnet-runtime",
					Version:  "2.2.5",
					Metadata: buildplan.Metadata{"build": true, "launch": true},
				}},
			}))
		})
		it("passes when we can find a compatible sdk for the given aspnet and there is an executable that with the same name as the app", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
  "runtimeOptions": {
    "tfm": "netcoreapp2.2",
    "framework": {
      "name": "Microsoft.AspNetCore.App",
      "version": "2.2.5"
    }
  }
}
`), os.ModePerm)).To(Succeed())
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: sdk.DotnetSDK}},
				Requires: []buildplan.Required{{
					Name:     sdk.DotnetSDK,
					Version:  "2.2.5",
					Metadata: buildplan.Metadata{"build": true, "launch": true},
				}, {
					Name:     "dotnet-runtime",
					Version:  "2.2.5",
					Metadata: buildplan.Metadata{"build": true, "launch": true},
				}, {
					Name:     "dotnet-aspnetcore",
					Version:  "2.2.5",
					Metadata: buildplan.Metadata{"build": true, "launch": true},
				}},
			}))
		})

	})

	when("the app is a SCD", func() {
		it("return with just a provides", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
  "runtimeOptions": {}
}
`), os.ModePerm)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(factory.Detect.Application.Root, "appName"), []byte(`fake exe`), os.ModePerm)).To(Succeed())
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: sdk.DotnetSDK}},
			}))
		})
	})

	when("the app is source based", func() {
		it("return with just a provides", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: sdk.DotnetSDK}},
			}))
		})
	})

	when("the app is a FDD with a FDE", func() {
		it("return with just a provides", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
  "runtimeOptions": {
    "tfm": "netcoreapp2.2",
    "framework": {
      "name": "Microsoft.NETCore.App",
      "version": "2.2.5"
    }
  }
}
`), os.ModePerm)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(factory.Detect.Application.Root, "appName"), []byte(`fake exe`), os.ModePerm)).To(Succeed())
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: sdk.DotnetSDK}},
			}))
		})
	})
}
