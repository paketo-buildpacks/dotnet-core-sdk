package dotnetcoresdk_test

import (
	"os"
	"path/filepath"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testGlobalFileParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("FindGlobalJson", func() {
		it("parses a valid global.json file", func() {
			tempDir := t.TempDir()
			globalJsonPath := filepath.Join(tempDir, "global.json")
			err := os.WriteFile(globalJsonPath, []byte(`{
				"sdk": {
					"version": "8.0.100",
					"rollForward": "latestPatch",
					"allowPrerelease": false
				}
			}`), 0644)
			Expect(err).NotTo(HaveOccurred())

			globalJson, err := dotnetcoresdk.FindGlobalJson(tempDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(globalJson).NotTo(BeNil())
			Expect(globalJson.Sdk).NotTo(BeNil())
			Expect(*globalJson.Sdk.Version).To(Equal("8.0.100"))
			Expect(*globalJson.Sdk.RollForward).To(Equal("latestPatch"))
			Expect(*globalJson.Sdk.AllowPrerelease).To(BeFalse())
		})

		it("finds global.json in parent directories", func() {
			tempDir := t.TempDir()
			subDirs := filepath.Join(tempDir, "subdir1", "subdir2")
			err := os.MkdirAll(subDirs, 0755)
			Expect(err).NotTo(HaveOccurred())

			globalJsonPath := filepath.Join(tempDir, "global.json")
			err = os.WriteFile(globalJsonPath, []byte(`{
				"sdk": {
					"version": "7.0.200"
				}
			}`), 0644)
			Expect(err).NotTo(HaveOccurred())

			globalJson, err := dotnetcoresdk.FindGlobalJson(subDirs)
			Expect(err).NotTo(HaveOccurred())
			Expect(globalJson).NotTo(BeNil())
			Expect(globalJson.Sdk).NotTo(BeNil())
			Expect(*globalJson.Sdk.Version).To(Equal("7.0.200"))
		})

		it("returns nil if no global.json is found", func() {
			tempDir := t.TempDir()

			globalJson, err := dotnetcoresdk.FindGlobalJson(tempDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(globalJson).To(BeNil())
		})

		it("returns closest global.json if multiple exist", func() {
			tempDir := t.TempDir()
			subDirs := filepath.Join(tempDir, "subdir1", "subdir2")
			err := os.MkdirAll(subDirs, 0755)
			Expect(err).NotTo(HaveOccurred())

			globalJsonPath1 := filepath.Join(tempDir, "global.json")
			err = os.WriteFile(globalJsonPath1, []byte(`{
				"sdk": {
					"version": "6.0.300"
				}
			}`), 0644)
			Expect(err).NotTo(HaveOccurred())

			globalJsonPath2 := filepath.Join(subDirs, "global.json")
			err = os.WriteFile(globalJsonPath2, []byte(`{
				"sdk": {
					"version": "7.0.400"
				}
			}`), 0644)
			Expect(err).NotTo(HaveOccurred())

			globalJson, err := dotnetcoresdk.FindGlobalJson(subDirs)
			Expect(err).NotTo(HaveOccurred())
			Expect(globalJson).NotTo(BeNil())
			Expect(globalJson.Sdk).NotTo(BeNil())
			Expect(*globalJson.Sdk.Version).To(Equal("7.0.400"))
		})

		it("returns an error for invalid global.json", func() {
			tempDir := t.TempDir()
			globalJsonPath := filepath.Join(tempDir, "global.json")
			err := os.WriteFile(globalJsonPath, []byte(`{ invalid json }`), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = dotnetcoresdk.FindGlobalJson(tempDir)
			Expect(err).To(MatchError(ContainSubstring("failed to parse global.json")))
		})
	})

	context("GetConstraintsFromGlobalJson", func() {
		it("generates exact version constraint", func() {
			globalJson := dotnetcoresdk.GlobalJson{
				Sdk: &dotnetcoresdk.Sdk{
					Version:     ptr("5.0.201"),
					RollForward: ptr("disabled"),
				},
			}

			constraints, err := dotnetcoresdk.GetConstraintsFromGlobalJson(globalJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]dotnetcoresdk.ConstraintResult{
				{
					Constraint: "5.0.201",
					Name:       "global.json exact",
				},
			}))
		})

		it("generates constraints for next version rollForward", func() {
			globalJson := dotnetcoresdk.GlobalJson{
				Sdk: &dotnetcoresdk.Sdk{
					Version:     ptr("6.0.205"),
					RollForward: ptr("major"),
				},
			}

			constraints, err := dotnetcoresdk.GetConstraintsFromGlobalJson(globalJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]dotnetcoresdk.ConstraintResult{
				{
					Constraint: ">= 6.0.205, < 6.0.300",
					Name:       "global.json patch",
				},
				{
					Constraint: ">= 6.0.300, < 6.0.400",
					Name:       "global.json feature",
				},
				{
					Constraint: ">= 6.1.100, < 6.1.200",
					Name:       "global.json minor",
				},
				{
					Constraint: ">= 7.0.100, < 7.0.200",
					Name:       "global.json major",
				},
			}))
		})

		it("generates constraints for latest feature rollForward", func() {
			globalJson := dotnetcoresdk.GlobalJson{
				Sdk: &dotnetcoresdk.Sdk{
					Version:     ptr("7.0.150"),
					RollForward: ptr("latestFeature"),
				},
			}

			constraints, err := dotnetcoresdk.GetConstraintsFromGlobalJson(globalJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]dotnetcoresdk.ConstraintResult{
				{
					Constraint: ">= 7.0.150, 7.0.*",
					Name:       "global.json feature",
				},
			}))
		})

		it("generates constraints for latest minor rollForward", func() {
			globalJson := dotnetcoresdk.GlobalJson{
				Sdk: &dotnetcoresdk.Sdk{
					Version:     ptr("8.1.300"),
					RollForward: ptr("latestMinor"),
				},
			}
			constraints, err := dotnetcoresdk.GetConstraintsFromGlobalJson(globalJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]dotnetcoresdk.ConstraintResult{
				{
					Constraint: ">= 8.1.300, 8.*.*",
					Name:       "global.json minor",
				},
			}))
		})

		it("generates constraints for latest major rollForward", func() {
			globalJson := dotnetcoresdk.GlobalJson{
				Sdk: &dotnetcoresdk.Sdk{
					Version:     ptr("9.2.400"),
					RollForward: ptr("latestMajor"),
				},
			}
			constraints, err := dotnetcoresdk.GetConstraintsFromGlobalJson(globalJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]dotnetcoresdk.ConstraintResult{
				{
					Constraint: ">= 9.2.400",
					Name:       "global.json major",
				},
			}))
		})
	})
}

func ptr[T any](v T) *T {
	return &v
}
