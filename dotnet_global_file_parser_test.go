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

	context("GetRollforwardConstraints", func() {
		it("generates exact version constraint", func() {
			constraints, err := dotnetcoresdk.GetRollforwardConstraints("5.0.201", "disabled")
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]string{
				"5.0.201",
			}))
		})

		it("generates constraints for next version rollForward", func() {
			constraints, err := dotnetcoresdk.GetRollforwardConstraints("6.0.205", "major")
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]string{
				">= 6.0.205, < 6.0.300",
				">= 6.0.300, < 6.0.400",
				">= 6.1.100, < 6.1.200",
				">= 7.0.100, < 7.0.200",
			}))
		})

		it("generates constraints for latest feature rollForward", func() {
			constraints, err := dotnetcoresdk.GetRollforwardConstraints("7.0.150", "latestFeature")
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]string{
				">= 7.0.150, 7.0.*",
			}))
		})

		it("generates constraints for latest minor rollForward", func() {
			constraints, err := dotnetcoresdk.GetRollforwardConstraints("8.1.300", "latestMinor")
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]string{
				">= 8.1.300, 8.*.*",
			}))
		})

		it("generates constraints for latest major rollForward", func() {
			constraints, err := dotnetcoresdk.GetRollforwardConstraints("9.2.400", "latestMajor")
			Expect(err).NotTo(HaveOccurred())
			Expect(constraints).To(Equal([]string{
				">= 9.2.400",
			}))
		})
	})
}
