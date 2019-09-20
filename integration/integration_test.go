package integration_test

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"io/ioutil"
	"path/filepath"
"testing"


"github.com/cloudfoundry/dagger"

"github.com/sclevine/spec"
"github.com/sclevine/spec/report"

. "github.com/onsi/gomega"
)

var (
	bpDir, aspnetURI, runtimeURI, sdkURI string
)

var suite = spec.New("Integration", spec.Report(report.Terminal{}))

func init() {
	suite("Integration", testIntegration)
}

func TestIntegration(t *testing.T) {
	var err error
	Expect := NewWithT(t).Expect
	bpDir, err = dagger.FindBPRoot()
	Expect(err).NotTo(HaveOccurred())

	sdkURI, err = dagger.PackageBuildpack(bpDir)
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(sdkURI)

	aspnetURI, err = dagger.GetLatestBuildpack("dotnet-core-aspnet-cnb")
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(aspnetURI)

	runtimeURI, err = dagger.GetLatestBuildpack("dotnet-core-runtime-cnb")
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(runtimeURI)

	suite.Run(t)
}

func testIntegration(t *testing.T, _ spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		Eventually func(interface{}, ...interface{}) AsyncAssertion
		app    *dagger.App
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually
	})

	it.After(func() {
		if app != nil {
			app.Destroy()
		}
	})



	it("should build a working OCI image for a simple app with aspnet dependencies", func() {
		app, err := dagger.PackBuild(filepath.Join("testdata", "simple_web_app"), runtimeURI, aspnetURI, sdkURI)
		Expect(err).ToNot(HaveOccurred())

		Expect(app.StartWithCommand("dotnet simple_web_app.dll --server.urls http://0.0.0.0:${PORT}")).To(Succeed())

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("Welcome"))
	})

	it("should build a working OCI image for a simple app with aspnet dependencies", func() {
		majorMinor := "2.2"
		version, err := getLowestRuntimeVersionInMajorMinor(majorMinor)
		Expect(err).ToNot(HaveOccurred())
		glbJson := fmt.Sprintf(`{
"sdk": { "version": "%s"}
}
`, version)

		glbJsonPath := filepath.Join("testdata", "simple_web_app_with_global_json", "global.json")
		Expect(ioutil.WriteFile(glbJsonPath, []byte(glbJson), 0644)).To(Succeed())

		app, err := dagger.PackBuild(filepath.Join("testdata", "simple_web_app_with_global_json"), runtimeURI, aspnetURI, sdkURI)
		Expect(err).ToNot(HaveOccurred())

		Expect(app.StartWithCommand("dotnet simple_web_app.dll --server.urls http://0.0.0.0:${PORT}")).To(Succeed())

		Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("dotnet-sdk.%s", version)))

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("Welcome"))
	})

	it("should build a working OCI image for a simple app with aspnet dependencies", func() {
		majorMinor := "2.2"
		version, err := getLowestRuntimeVersionInMajorMinor(majorMinor)
		Expect(err).ToNot(HaveOccurred())
		bpYml := fmt.Sprintf(`---
dotnet-sdk:
  version: "%s"
`, version)

		bpYmlPath := filepath.Join("testdata", "simple_web_app_with_buildpack_yml", "buildpack.yml")
		Expect(ioutil.WriteFile(bpYmlPath, []byte(bpYml), 0644)).To(Succeed())

		app, err := dagger.PackBuild(filepath.Join("testdata", "simple_web_app_with_buildpack_yml"), runtimeURI, aspnetURI, sdkURI)
		Expect(err).ToNot(HaveOccurred())

		Expect(app.StartWithCommand("dotnet simple_web_app.dll --server.urls http://0.0.0.0:${PORT}")).To(Succeed())

		Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("dotnet-sdk.%s", version)))

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("Welcome"))
	})

	it("should build a working OCI image for a simple app with aspnet dependencies", func() {
		glbJson := `{
"sdk": { "version": "2.2.100"}
}`

		glbJsonPath := filepath.Join("testdata", "simple_web_app_with_buildpack_yml_and_global_json", "global.json")
		Expect(ioutil.WriteFile(glbJsonPath, []byte(glbJson), 0644)).To(Succeed())

		majorMinor := "2.2"
		version, err := getLowestRuntimeVersionInMajorMinor(majorMinor)
		Expect(err).ToNot(HaveOccurred())
		bpYml := fmt.Sprintf(`---
dotnet-sdk:
  version: "%s"
`, version)

		bpYmlPath := filepath.Join("testdata", "simple_web_app_with_buildpack_yml_and_global_json", "buildpack.yml")
		Expect(ioutil.WriteFile(bpYmlPath, []byte(bpYml), 0644)).To(Succeed())

		app, err := dagger.PackBuild(filepath.Join("testdata", "simple_web_app_with_buildpack_yml_and_global_json"), runtimeURI, aspnetURI, sdkURI)
		Expect(err).ToNot(HaveOccurred())

		Expect(app.StartWithCommand("dotnet simple_web_app.dll --server.urls http://0.0.0.0:${PORT}")).To(Succeed())

		Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("dotnet-sdk.%s", version)))

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("Welcome"))


	})
}

func getLowestRuntimeVersionInMajorMinor(majorMinor string) (string, error) {
	type buildpackTomlVersion struct {
		Metadata struct {
			Dependencies []struct {
				Version string `toml:"version"`
			} `toml:"dependencies"`
		} `toml:"metadata"`
	}

	bpToml := buildpackTomlVersion{}
	output, err := ioutil.ReadFile(filepath.Join("..", "buildpack.toml"))
	if err != nil {
		return "", err
	}

	majorMinorConstraint, err := semver.NewConstraint(fmt.Sprintf("%s.*", majorMinor))
	if err != nil {
		return "", err
	}

	lowestVersion, err := semver.NewVersion(fmt.Sprintf("%s.9999", majorMinor))
	if err != nil {
		return "", err
	}

	_, err = toml.Decode(string(output), &bpToml)
	if err != nil {
		return "", err
	}

	for _, dep := range bpToml.Metadata.Dependencies {
		depVersion, err := semver.NewVersion(dep.Version)
		if err != nil {
			return "", err
		}
		if majorMinorConstraint.Check(depVersion){
			if depVersion.LessThan(lowestVersion){
				lowestVersion = depVersion
			}
		}
	}

	return lowestVersion.String(), nil
}

