package sdk_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/dotnet-core-sdk/sdk"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitRuntimeConfig(t *testing.T) {
	spec.Run(t, "Runtime Config", testRuntimeConfig, spec.Report(report.Terminal{}))
}

func testRuntimeConfig(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("when there is a valid runtimeconfig.json and the framework given is aspnet", func() {
		it("parses", func() {
			appRoot, err := ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())
			runtimeConfigJSONPath := filepath.Join(appRoot, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
  "runtimeOptions": {
    "tfm": "netcoreapp2.2",
    "framework": {
      "name": "Microsoft.AspNetCore.App",
      "version": "2.2.5"
	},
    "applyPatches": true
  }
}
`), os.ModePerm)).To(Succeed())
			defer os.RemoveAll(appRoot)
			runtimeConfig, err := sdk.NewRuntimeConfig(appRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(runtimeConfig.HasASPNetDependency()).To(BeTrue())
			Expect(runtimeConfig.HasApplyPatches()).To(BeTrue())

		})

		it("parses when comments are in runtime.json", func() {
			appRoot, err := ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())
			runtimeConfigJSONPath := filepath.Join(appRoot, "appName.runtimeconfig.json")

			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
  "runtimeOptions": {
    /*
    Multi line
    Comment
    */
    "tfm": "netcoreapp2.2",
    "framework": {
	  "name": "Microsoft.AspNetCore.All",
	  "version": "2.2.5"
    },
    // comment here ok?
    "configProperties": {
	  "System.GC.Server": true
    }
  }
}
		`), os.ModePerm)).To(Succeed())
			defer os.RemoveAll(appRoot)
			runtimeConfig, err := sdk.NewRuntimeConfig(appRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(runtimeConfig.HasASPNetDependency()).To(BeTrue())
			Expect(runtimeConfig.HasApplyPatches()).To(BeFalse())
		})
	})

	when("when there are multiple runtimeconfig.json", func() {
		it("fails fast", func() {
			appRoot, err := ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())
			runtimeConfigJSONPath := filepath.Join(appRoot, "appName.runtimeconfig.json")
			anotherRuntimeConfigJSONPath := filepath.Join(appRoot, "another.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`{}`), os.ModePerm)).To(Succeed())
			Expect(ioutil.WriteFile(anotherRuntimeConfigJSONPath, []byte(`{}`), os.ModePerm)).To(Succeed())
			defer os.RemoveAll(appRoot)

			runtimeConfig, err := sdk.NewRuntimeConfig(appRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("multiple *.runtimeconfig.json files present"))
			Expect(runtimeConfig.HasASPNetDependency()).To(BeFalse())
		})
	})

	when("there is not runtimeconfig.json at the given root", func() {
		it("the runtime detector returns false", func() {
			appRoot, err := ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(appRoot)

			runtimeConfig, err := sdk.NewRuntimeConfig(appRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(runtimeConfig.IsPresent()).To(BeFalse())
			Expect(runtimeConfig.HasASPNetDependency()).To(BeFalse())
		})

	})

	when("the app is FDD and", func() {
		it("has an executable, the detector returns true", func() {
			appRoot, err := ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())
			runtimeConfigJSONPath := filepath.Join(appRoot, "appName.runtimeconfig.json")
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
			Expect(ioutil.WriteFile(filepath.Join(appRoot, "appName"), []byte(`fake exe`), os.ModePerm)).To(Succeed())
			defer os.RemoveAll(appRoot)

			runtimeConfig, err := sdk.NewRuntimeConfig(appRoot)
			Expect(err).ToNot(HaveOccurred())

			hasExecutable, err := runtimeConfig.HasExecutable()
			Expect(err).ToNot(HaveOccurred())
			Expect(hasExecutable).To(BeTrue())
		})

		it("does not have an executable, the detector returns false", func() {
			appRoot, err := ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())
			runtimeConfigJSONPath := filepath.Join(appRoot, "appName.runtimeconfig.json")
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
			defer os.RemoveAll(appRoot)

			runtimeConfig, err := sdk.NewRuntimeConfig(appRoot)
			Expect(err).ToNot(HaveOccurred())

			hasExecutable, err := runtimeConfig.HasExecutable()
			Expect(err).ToNot(HaveOccurred())
			Expect(hasExecutable).To(BeFalse())
		})
	})

}
