package sdk

import (
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitVersioning(t *testing.T) {
	spec.Run(t, "Versioning", testVersioning, spec.Report(report.Terminal{}))
}

func testVersioning(t *testing.T, when spec.G, it spec.S) {

	when("GetLatestCompatibleSDKConstraint", func() {
		it("returns a patch constrained version", func() {
			version, err := GetLatestCompatibleSDKConstraint("2.2.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(version).To(Equal("2.2.*"))
		})
	})

	when("IsCompatibleSDKOptionWithRuntime", func() {
		when("the sdk version is compatible with the runtime verion", func() {
			it("returns true and no error when the feature line and patch are a wildcard", func() {
				compatible, err := IsCompatibleSDKOptionWithRuntime("2.2.*", "2.2.*")
				Expect(err).ToNot(HaveOccurred())
				Expect(compatible).To(BeTrue())
			})

			it("returns true and no error when the patch is a wildcard", func() {
				compatible, err := IsCompatibleSDKOptionWithRuntime("2.2.*", "2.2.1*")
				Expect(err).ToNot(HaveOccurred())
				Expect(compatible).To(BeTrue())
			})

			it("returns true and no error when the version is set", func() {
				compatible, err := IsCompatibleSDKOptionWithRuntime("2.2.*", "2.2.100")
				Expect(err).ToNot(HaveOccurred())
				Expect(compatible).To(BeTrue())
			})
		})

		when("the sdk version is not compatible with the runtime verion", func() {
			it("returns true and no error when the feature line and patch are a wildcard", func() {
				compatible, err := IsCompatibleSDKOptionWithRuntime("2.2.*", "2.3.*")
				Expect(err).ToNot(HaveOccurred())
				Expect(compatible).To(BeFalse())
			})

			it("returns true and no error when the patch is a wildcard", func() {
				compatible, err := IsCompatibleSDKOptionWithRuntime("2.2.*", "2.3.1*")
				Expect(err).ToNot(HaveOccurred())
				Expect(compatible).To(BeFalse())
			})

			it("returns true and no error when the version is set", func() {
				compatible, err := IsCompatibleSDKOptionWithRuntime("2.2.*", "2.3.100")
				Expect(err).ToNot(HaveOccurred())
				Expect(compatible).To(BeFalse())
			})
		})
	})

	when("GetConstrainedCompatibleSDK", func() {
		var (
			factory                 *test.BuildFactory
			stubDotnetSDKFixture    = filepath.Join("testdata", "stub-sdk-dependency.tar.xz")
		)

		it.Before(func() {
			RegisterTestingT(t)
			factory = test.NewBuildFactory(t)
			factory.AddDependencyWithVersion(DotnetSDK, "2.2.805", stubDotnetSDKFixture)
			factory.AddDependencyWithVersion(DotnetSDK, "2.2.605", stubDotnetSDKFixture)
		})

		when("a compatible version of the sdk is present", func() {
			when("the feature line and patch are a wildcard", func() {
				it("returns the latest sdk for the runtime constraint", func() {
					version, err := GetConstrainedCompatibleSDK("2.2.*", factory.Build)
					Expect(err).ToNot(HaveOccurred())
					Expect(version).To(Equal("2.2.805"))
				})
			})

			when("the patch is a wildcard", func() {
				it("returns the latest sdk for the runtime and feature line constraint", func() {
					version, err := GetConstrainedCompatibleSDK("2.2.6*", factory.Build)
					Expect(err).ToNot(HaveOccurred())
					Expect(version).To(Equal("2.2.605"))
				})
			})

			when("the exact version", func() {
				it("returns the latest sdk for the runtime and feature line constraint", func() {
					factory.AddDependencyWithVersion(DotnetSDK, "2.2.804", stubDotnetSDKFixture)
					version, err := GetConstrainedCompatibleSDK("2.2.804", factory.Build)
					Expect(err).ToNot(HaveOccurred())
					Expect(version).To(Equal("2.2.804"))
				})
			})
		})

		when("there are no compatible versions of the sdk", func() {
			when("the feature line and patch are a wildcard", func() {
				it("returns an error message and an empty string", func() {
					version, err := GetConstrainedCompatibleSDK("2.3.*", factory.Build)
					Expect(err).To(Equal(fmt.Errorf("no sdk version matching 2.3.* found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version")))
					Expect(version).To(Equal(""))
				})
			})

			when("the patch is a wildcard", func() {
				it("returns an error message and an empty string", func() {
					version, err := GetConstrainedCompatibleSDK("2.2.1*", factory.Build)
					Expect(err).To(Equal(fmt.Errorf("no sdk version matching 2.2.1* found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version")))
					Expect(version).To(Equal(""))
				})
			})

			when("the exact version", func() {
				it("returns the latest sdk for the runtime and feature line constraint", func() {
					version, err := GetConstrainedCompatibleSDK("2.2.804", factory.Build)
					Expect(err).To(Equal(fmt.Errorf("no sdk version matching 2.2.804 found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version")))
					Expect(version).To(Equal(""))
				})
			})
		})
	})

	when("GetFeatureLineConstraint", func() {
		it("return a feature line constraint when given a full sdk version", func() {
			versionConstraint, err := GetFeatureLineConstraint("2.2.800")
			Expect(err).ToNot(HaveOccurred())
			Expect(versionConstraint).To(ContainSubstring("2.2.8*"))
		})
	})

	when("GetConstrainedCompatibleSDKForGlobalJson", func() {
		var (
			factory                 *test.BuildFactory
			stubDotnetSDKFixture    = filepath.Join("testdata", "stub-sdk-dependency.tar.xz")
		)

		it.Before(func() {
			RegisterTestingT(t)
			factory = test.NewBuildFactory(t)
			factory.AddDependencyWithVersion(DotnetSDK, "2.2.805", stubDotnetSDKFixture)
			factory.AddDependencyWithVersion(DotnetSDK, "2.2.605", stubDotnetSDKFixture)
		})

		when("a compatible version of the sdk is present", func() {
			when("the feature line and patch specified in global.json are present", func() {
				it("returns the latest sdk for the runtime constraint", func() {
					version, err := GetConstrainedCompatibleSDKForGlobalJson("2.2.805", factory.Build)
					Expect(err).ToNot(HaveOccurred())
					Expect(version).To(Equal("2.2.805"))
				})
			})

			when("the patch cannot be found but there is a matching feature line", func() {
				it("returns the latest sdk for the runtime and feature line constraint", func() {
					version, err := GetConstrainedCompatibleSDKForGlobalJson("2.2.800", factory.Build)
					Expect(err).ToNot(HaveOccurred())
					Expect(version).To(Equal("2.2.805"))
				})
			})

		})

		when("there are no compatible versions of the sdk", func() {
			when("mismatch major/minor", func() {
				it("returns an error message and an empty string", func() {
					version, err := GetConstrainedCompatibleSDK("2.3.100", factory.Build)
					Expect(err).To(Equal(fmt.Errorf("no sdk version matching 2.3.100 found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version")))
					Expect(version).To(Equal(""))
				})
			})

			when("no matching feature line", func() {
				it("returns an error message and an empty string", func() {
					version, err := GetConstrainedCompatibleSDK("2.2.100", factory.Build)
					Expect(err).To(Equal(fmt.Errorf("no sdk version matching 2.2.100 found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version")))
					Expect(version).To(Equal(""))
				})
			})

			when("feature line patch is higher than the highest patch in buildpack.toml", func() {
				it("returns an error message and an empty string", func() {
					version, err := GetConstrainedCompatibleSDK("2.2.606", factory.Build)
					Expect(err).To(Equal(fmt.Errorf("no sdk version matching 2.2.606 found, please reconfigure the global.json and/or buildpack.yml to use supported sdk version")))
					Expect(version).To(Equal(""))
				})
			})
		})
	})

	when("SelectRollStrategy", func() {
		when("buildpack.yml has a version without a wildcard in it", func() {
			when("the buildpack.yml version is the same as the one in global.json", func() {
				it("returns true to use buildpack.yml version, false for global.json, and no error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("2.2.100", "2.2.100")
					Expect(err).ToNot(HaveOccurred())
					Expect(useBuildpackYAML).To(BeTrue())
					Expect(useGlobalJSON).To(BeFalse())
				})
			})

			when("the buildpack.yml version is compatible with the one in global.json", func() {
				it("returns true to use buildpack.yml version, false for global.json, and no error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("2.2.104", "2.2.100")
					Expect(err).ToNot(HaveOccurred())
					Expect(useBuildpackYAML).To(BeTrue())
					Expect(useGlobalJSON).To(BeFalse())
				})
			})

			when("the buildpack.yml version's patch is lower with the global.json version's patch", func() {
				it("returns false to use buildpack.yml version, false for global.json, and an error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("2.2.100", "2.2.104")
					Expect(err).To(Equal(fmt.Errorf(IncompatibleGlobalAndBuildpackYml)))
					Expect(useBuildpackYAML).To(BeFalse())
					Expect(useGlobalJSON).To(BeFalse())
				})
			})

			when("the buildpack.yml version's minor does not match the global.json version's minor", func() {
				it("returns false to use buildpack.yml version, false for global.json, and an error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("2.3.100", "2.2.100")
					Expect(err).To(Equal(fmt.Errorf(IncompatibleGlobalAndBuildpackYml)))
					Expect(useBuildpackYAML).To(BeFalse())
					Expect(useGlobalJSON).To(BeFalse())
				})
			})

			when("the buildpack.yml version's major does not match the global.json version's major", func() {
				it("returns false to use buildpack.yml version, false for global.json, and an error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("3.2.100", "2.2.100")
					Expect(err).To(Equal(fmt.Errorf(IncompatibleGlobalAndBuildpackYml)))
					Expect(useBuildpackYAML).To(BeFalse())
					Expect(useGlobalJSON).To(BeFalse())
				})
			})
		})

		when("the buildpack.yml version is a constraint", func() {
			when("the feature line in buildpack.yml and global.json are the same", func(){
				it("returns false for buildpack.yml, true for global.json, and no error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("2.2.1*", "2.2.102")
					Expect(err).ToNot(HaveOccurred())
					Expect(useBuildpackYAML).To(BeFalse())
					Expect(useGlobalJSON).To(BeTrue())
				})
			})

			when("the patch  in buildpack.yml and global.json are the same", func(){
				it("returns false for buildpack.yml, true for global.json, and no error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("2.2.*", "2.2.102")
					Expect(err).ToNot(HaveOccurred())
					Expect(useBuildpackYAML).To(BeFalse())
					Expect(useGlobalJSON).To(BeTrue())
				})
			})

			when("the feature line in buildpack.yml and global.json are not the same", func(){
				it("returns false for buildpack.yml, true for global.json, and no error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("2.2.2*", "2.2.102")
					Expect(err).To(Equal(fmt.Errorf(IncompatibleGlobalAndBuildpackYml)))
					Expect(useBuildpackYAML).To(BeFalse())
					Expect(useGlobalJSON).To(BeFalse())
				})
			})

			when("the minor in buildpack.yml and global.json are not the same", func(){
				it("returns false for buildpack.yml, true for global.json, and no error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("2.1.*", "2.2.102")
					Expect(err).To(Equal(fmt.Errorf(IncompatibleGlobalAndBuildpackYml)))
					Expect(useBuildpackYAML).To(BeFalse())
					Expect(useGlobalJSON).To(BeFalse())
				})
			})

			when("the major in buildpack.yml and global.json are not the same", func(){
				it("returns false for buildpack.yml, true for global.json, and no error", func() {
					useBuildpackYAML, useGlobalJSON, err := SelectRollStrategy("3.1.*", "2.2.102")
					Expect(err).To(Equal(fmt.Errorf(IncompatibleGlobalAndBuildpackYml)))
					Expect(useBuildpackYAML).To(BeFalse())
					Expect(useGlobalJSON).To(BeFalse())
				})
			})
		})
	})
}
