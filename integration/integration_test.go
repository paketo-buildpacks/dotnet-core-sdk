package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	sdkURI, builder string
	bpList          []string
)

const testBuildpack = "test-buildpack"

var suite = spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())

func init() {
	suite("Integration", testIntegration)
}

func Package(root, version string, cached bool) (string, error) {
	var cmd *exec.Cmd

	bpPath := filepath.Join(root, "artifact")
	if cached {
		cmd = exec.Command(".bin/packager", "--archive", "--version", version, fmt.Sprintf("%s-cached", bpPath))
	} else {
		cmd = exec.Command(".bin/packager", "--archive", "--uncached", "--version", version, bpPath)
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("PACKAGE_DIR=%s", bpPath))
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if cached {
		return fmt.Sprintf("%s-cached.tgz", bpPath), err
	}

	return fmt.Sprintf("%s.tgz", bpPath), err
}

func BeforeSuite() {
	root, err := filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	sdkURI, err = Package(root, "1.2.3", false)
	Expect(err).NotTo(HaveOccurred())

	config, err := dagger.ParseConfig("config.json")
	Expect(err).NotTo(HaveOccurred())

	builder = config.Builder

	for _, bp := range config.BuildpackOrder[builder] {
		var bpURI string
		if bp == testBuildpack {
			bpList = append(bpList, sdkURI)
			continue
		}
		bpURI, err = dagger.GetLatestCommunityBuildpack("paketo-buildpacks", bp)
		Expect(err).NotTo(HaveOccurred())
		bpList = append(bpList, bpURI)
	}
}

func AfterSuite() {
	for _, bp := range bpList {
		Expect(dagger.DeleteBuildpack(bp)).To(Succeed())
	}
}

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)
	BeforeSuite()
	suite.Run(t)
	AfterSuite()
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect     func(interface{}, ...interface{}) Assertion
		Eventually func(interface{}, ...interface{}) AsyncAssertion
		app        *dagger.App
		err        error
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually
	})

	it.After(func() {
		if app != nil {
			Expect(app.Destroy()).To(Succeed())
		}
	})

	it("should build a working OCI image for a simple app with aspnet dependencies", func() {
		app, err = dagger.NewPack(
			filepath.Join("testdata", "simple_web_app_3.1"),
			dagger.RandomImage(),
			dagger.SetBuildpacks(bpList...),
			dagger.SetBuilder(builder),
		).Build()
		Expect(err).ToNot(HaveOccurred())

		if builder == "bionic" {
			app.SetHealthCheck("stat /workspace", "2s", "15s")
		}

		Expect(app.StartWithCommand("dotnet simple_web_app.dll --urls http://0.0.0.0:${PORT}")).To(Succeed())

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("Hello World!"))
	})

	when("global.json is specified", func() {
		it("should build a working OCI image for a simple app with aspnet dependencies", func() {
			majorMinor := "3.1"
			version, err := GetLowestRuntimeVersionInMajorMinor(majorMinor, filepath.Join("..", "buildpack.toml"))
			Expect(err).ToNot(HaveOccurred())
			glbJson := fmt.Sprintf(`{
"sdk": { "version": "%s"}
}
`, version)

			glbJsonPath := filepath.Join("testdata", "simple_web_app_with_global_json_3.1", "global.json")
			Expect(ioutil.WriteFile(glbJsonPath, []byte(glbJson), 0644)).To(Succeed())

			app, err = dagger.NewPack(
				filepath.Join("testdata", "simple_web_app_with_global_json_3.1"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(bpList...),
				dagger.SetBuilder(builder),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			if builder == "bionic" {
				app.SetHealthCheck("stat /workspace", "2s", "15s")
			}

			Expect(app.StartWithCommand("dotnet simple_web_app.dll --urls http://0.0.0.0:${PORT}")).To(Succeed())

			Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("dotnet-sdk_%s", version)))

			Eventually(func() string {
				body, _, _ := app.HTTPGet("/")
				return body
			}).Should(ContainSubstring("Hello World!"))
		})
	})

	when("buildpack.yml is specified", func() {
		it("should build a working OCI image for a simple app with aspnet dependencies", func() {
			majorMinor := "3.1"
			version, err := GetLowestRuntimeVersionInMajorMinor(majorMinor, filepath.Join("..", "buildpack.toml"))
			Expect(err).ToNot(HaveOccurred())
			frameworkVersion, err := GetCorrespondingRuntimeFromSDK(version, filepath.Join("..", "buildpack.toml"))
			Expect(err).ToNot(HaveOccurred())

			bpYml := fmt.Sprintf(`---
dotnet-framework:
  version: "%s"
dotnet-sdk:
  version: "%s"
`, frameworkVersion, version)

			bpYmlPath := filepath.Join("testdata", "simple_web_app_with_buildpack_yml_3.1", "buildpack.yml")
			Expect(ioutil.WriteFile(bpYmlPath, []byte(bpYml), 0644)).To(Succeed())

			app, err = dagger.NewPack(
				filepath.Join("testdata", "simple_web_app_with_buildpack_yml_3.1"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(bpList...),
				dagger.SetBuilder(builder),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			if builder == "bionic" {
				app.SetHealthCheck("stat /workspace", "2s", "15s")
			}

			Expect(app.StartWithCommand("dotnet simple_web_app.dll --urls http://0.0.0.0:${PORT}")).To(Succeed())

			Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("dotnet-sdk_%s", version)))

			Eventually(func() string {
				body, _, _ := app.HTTPGet("/")
				return body
			}).Should(ContainSubstring("Hello World!"))
		})
	})

	when("buildpack.yml and global.json are specified", func() {
		it("should build a working OCI image for a simple app with aspnet dependencies", func() {
			glbJson := `{
"sdk": { "version": "3.1.102"}
}`

			glbJsonPath := filepath.Join("testdata", "simple_web_app_with_buildpack_yml_and_global_json_3.1", "global.json")
			Expect(ioutil.WriteFile(glbJsonPath, []byte(glbJson), 0644)).To(Succeed())

			majorMinor := "3.1"
			version, err := GetLowestRuntimeVersionInMajorMinor(majorMinor, filepath.Join("..", "buildpack.toml"))
			Expect(err).ToNot(HaveOccurred())
			frameworkVersion, err := GetCorrespondingRuntimeFromSDK(version, filepath.Join("..", "buildpack.toml"))
			Expect(err).ToNot(HaveOccurred())

			bpYml := fmt.Sprintf(`---
dotnet-framework:
  version: "%s"
dotnet-sdk:
  version: "%s"
`, frameworkVersion, version)

			bpYmlPath := filepath.Join("testdata", "simple_web_app_with_buildpack_yml_and_global_json_3.1", "buildpack.yml")
			Expect(ioutil.WriteFile(bpYmlPath, []byte(bpYml), 0644)).To(Succeed())

			app, err = dagger.NewPack(
				filepath.Join("testdata", "simple_web_app_with_buildpack_yml_and_global_json_3.1"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(bpList...),
				dagger.SetBuilder(builder),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			if builder == "bionic" {
				app.SetHealthCheck("stat /workspace", "2s", "15s")
			}

			Expect(app.StartWithCommand("dotnet simple_web_app.dll --urls http://0.0.0.0:${PORT}")).To(Succeed())

			Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("dotnet-sdk_%s", version)))

			Eventually(func() string {
				body, _, _ := app.HTTPGet("/")
				return body
			}).Should(ContainSubstring("Hello World!"))
		})
	})

	// TODO: Template this to make them less brittle
	it("should build a working OCI image for a fdd app with an old aspnet dependency that has not been rolled forward", func() {
		app, err = dagger.NewPack(
			filepath.Join("testdata", "fdd_apply_patches_false_2.1"),
			dagger.RandomImage(),
			dagger.SetBuildpacks(bpList...),
			dagger.SetBuilder(builder),
		).Build()

		Expect(err).ToNot(HaveOccurred())

		if builder == "bionic" {
			app.SetHealthCheck("stat /workspace", "2s", "15s")
		}

		Expect(app.StartWithCommand("dotnet dotnet.dll --urls http://0.0.0.0:${PORT}")).To(Succeed())

		Expect(app.BuildLogs()).To(MatchRegexp(fmt.Sprintf(`Dotnet Core Runtime %s`, `2\.1\.23`)))
		Expect(app.BuildLogs()).To(MatchRegexp(fmt.Sprintf(`Dotnet Core ASPNet %s`, `2\.1\.23`)))

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("dotnet"))
	})

	// TODO: template this to make it less brittle
	it("should build a working OCI image for a fdd app with an aspnet dependency that has been rolled forward", func() {
		app, err = dagger.NewPack(
			filepath.Join("testdata", "fdd_apply_patches_true_2.1"),
			dagger.RandomImage(),
			dagger.SetBuildpacks(bpList...),
			dagger.SetBuilder(builder),
		).Build()

		Expect(err).ToNot(HaveOccurred())

		if builder == "bionic" {
			app.SetHealthCheck("stat /workspace", "2s", "15s")
		}

		Expect(app.BuildLogs()).To(MatchRegexp(fmt.Sprintf(`Dotnet Core Runtime %s`, `2\.1\.24`)))
		Expect(app.BuildLogs()).To(MatchRegexp(fmt.Sprintf(`Dotnet Core ASPNet %s`, `2\.1\.24`)))

		Expect(app.StartWithCommand("dotnet dotnet.dll --urls http://0.0.0.0:${PORT}")).To(Succeed())

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("dotnet"))
	})

	it("should build a working OCI image for fdd asp vendored application", func() {
		app, err = dagger.NewPack(
			filepath.Join("testdata", "fdd_asp_vendored_2.1"),
			dagger.RandomImage(),
			dagger.SetBuildpacks(bpList...),
			dagger.SetBuilder(builder),
		).Build()
		Expect(err).ToNot(HaveOccurred())

		if builder == "bionic" {
			app.SetHealthCheck("stat /workspace", "2s", "15s")
		}

		Expect(app.StartWithCommand("dotnet asp_dotnet2.dll --urls http://0.0.0.0:${PORT}")).To(Succeed())

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("Hello World!"))

	})

	it("should build a working OCI image for fdd aspnet core application", func() {
		app, err = dagger.NewPack(
			filepath.Join("testdata", "fdd_aspnetcore_2.1"),
			dagger.RandomImage(),
			dagger.SetBuildpacks(bpList...),
			dagger.SetBuilder(builder),
		).Build()
		Expect(err).ToNot(HaveOccurred())

		if builder == "bionic" {
			app.SetHealthCheck("stat /workspace", "2s", "15s")
		}

		Expect(app.StartWithCommand("dotnet source_aspnetcore_2.1.dll --urls http://0.0.0.0:${PORT}")).To(Succeed())

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("Hello World!"))

	})

	it("should build a working OCI image for an application with comments in runtimeconfig", func() {
		app, err = dagger.NewPack(
			filepath.Join("testdata", "runtimeconfig_with_comments"),
			dagger.RandomImage(),
			dagger.SetBuildpacks(bpList...),
			dagger.SetBuilder(builder),
		).Build()
		Expect(err).ToNot(HaveOccurred())

		if builder == "bionic" {
			app.SetHealthCheck("stat /workspace", "2s", "15s")
		}

		Expect(app.StartWithCommand("dotnet source_aspnetcore_2.1.dll --urls http://0.0.0.0:${PORT}")).To(Succeed())

		Eventually(func() string {
			body, _, _ := app.HTTPGet("/")
			return body
		}).Should(ContainSubstring("Hello World!"))
	})
}

func GetLowestRuntimeVersionInMajorMinor(majorMinor, bpTOMLPath string) (string, error) {
	type buildpackTomlVersion struct {
		Metadata struct {
			Dependencies []struct {
				Version string `toml:"version"`
			} `toml:"dependencies"`
		} `toml:"metadata"`
	}

	bpToml := buildpackTomlVersion{}
	output, err := ioutil.ReadFile(filepath.Join(bpTOMLPath))
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
		if majorMinorConstraint.Check(depVersion) {
			if depVersion.LessThan(lowestVersion) {
				lowestVersion = depVersion
			}
		}
	}

	return lowestVersion.String(), nil
}

func GetCorrespondingRuntimeFromSDK(sdkVersion, bpTOMLPath string) (string, error) {
	var frameworkVersion string
	var runtimeSDKMap struct {
		Metadata struct {
			RuntimeToSdks []struct {
				RuntimeVersion string   `toml:"runtime-version"`
				Sdks           []string `toml:"sdks"`
			} `toml:"runtime-to-sdks"`
		} `toml:"metadata"`
	}

	_, err := toml.DecodeFile(bpTOMLPath, &runtimeSDKMap)
	if err != nil {
		return "", err
	}

	for _, r := range runtimeSDKMap.Metadata.RuntimeToSdks {
		for _, s := range r.Sdks {
			if s == sdkVersion {
				frameworkVersion = r.RuntimeVersion
			}
		}
	}
	if frameworkVersion == "" {
		return "", fmt.Errorf("no runtime version found for sdk-version %s", sdkVersion)
	}
	return frameworkVersion, nil
}
