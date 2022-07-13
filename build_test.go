package dotnetcoresdk_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/dotnet-core-sdk/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"

	//nolint Ignore SA1019, informed usage of deprecated package
	"github.com/paketo-buildpacks/packit/v2/paketosbom"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		cnbDir     string
		workingDir string
		buffer     *bytes.Buffer

		entryResolver     *fakes.EntryResolver
		dependencyManager *fakes.DependencyManager
		sbomGenerator     *fakes.SBOMGenerator

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error

		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "dotnet-sdk",
			Metadata: map[string]interface{}{
				"version":        "2.5.x",
				"version-source": "some-source",
			},
		}

		entryResolver.MergeLayerTypesCall.Returns.Build = true
		entryResolver.MergeLayerTypesCall.Returns.Launch = true

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:      "dotnet-sdk",
			Version: "some-version",
			Name:    ".NET Core SDK",
			SHA256:  "some-sha",
		}

		dependencyManager.DeliverCall.Stub = func(postal.Dependency, string, string, string) error {
			Expect(os.MkdirAll(filepath.Join(layersDir, "dotnet-core-sdk"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layersDir, "dotnet-core-sdk", "dotnet"), []byte(`hi`), os.ModePerm)).To(Succeed())
			return nil
		}

		dependencyManager.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "dotnet-sdk",
				Metadata: paketosbom.BOMMetadata{
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "dotnet-sdk-dep-sha",
					},
					Version: "dotnet-sdk-dep-version",
					URI:     "dotnet-sdk-dep-uri",
				},
			},
		}

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateFromDependencyCall.Returns.SBOM = sbom.SBOM{}

		buffer = bytes.NewBuffer(nil)

		build = dotnetcoresdk.Build(
			entryResolver,
			dependencyManager,
			sbomGenerator,
			scribe.NewEmitter(buffer),
			chronos.DefaultClock,
		)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that installs a version of the SDK into a layer", func() {
		result, err := build(packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Version:     "1.2.3",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "dotnet-sdk",
						Metadata: map[string]interface{}{
							"version-source": "some-source",
							"version":        "2.5.x",
							"build":          true,
							"launch":         true,
						},
					},
				},
			},
			Platform:   packit.Platform{Path: "platform"},
			Layers:     packit.Layers{Path: layersDir},
			CNBPath:    cnbDir,
			WorkingDir: workingDir,
			Stack:      "some-stack",
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		envLayer := result.Layers[0]
		SDKLayer := result.Layers[1]

		Expect(SDKLayer.Name).To(Equal("dotnet-core-sdk"))
		Expect(SDKLayer.BuildEnv).To(Equal(packit.Environment{
			"PATH.prepend": filepath.Join(layersDir, "dotnet-core-sdk"),
			"PATH.delim":   string(os.PathListSeparator),
		}))
		Expect(envLayer.LaunchEnv).To(Equal(packit.Environment{
			"PATH.prepend": filepath.Join(workingDir, ".dotnet_root"),
			"PATH.delim":   string(os.PathListSeparator),
		}))
		Expect(SDKLayer.Path).To(Equal(filepath.Join(layersDir, "dotnet-core-sdk")))
		Expect(SDKLayer.Metadata).To(Equal(map[string]interface{}{
			"dependency-sha": "some-sha",
		}))

		Expect(SDKLayer.Build).To(BeTrue())
		Expect(SDKLayer.Launch).To(BeTrue())
		Expect(SDKLayer.Cache).To(BeTrue())

		Expect(envLayer.Build).To(BeFalse())
		Expect(envLayer.Launch).To(BeTrue())
		Expect(envLayer.Cache).To(BeFalse())

		Expect(SDKLayer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
			{
				Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
			},
			{
				Extension: sbom.Format(sbom.SPDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
			},
		}))

		Expect(result.Build.BOM).To(HaveLen(1))
		buildBOMEntry := result.Build.BOM[0]
		Expect(buildBOMEntry.Name).To(Equal("dotnet-sdk"))
		Expect(buildBOMEntry.Metadata).To(Equal(paketosbom.BOMMetadata{
			Version: "dotnet-sdk-dep-version",
			Checksum: paketosbom.BOMChecksum{
				Algorithm: paketosbom.SHA256,
				Hash:      "dotnet-sdk-dep-sha",
			},
			URI: "dotnet-sdk-dep-uri",
		}))

		Expect(result.Launch.BOM).To(HaveLen(1))
		launchBOMEntry := result.Launch.BOM[0]
		Expect(launchBOMEntry.Name).To(Equal("dotnet-sdk"))
		Expect(launchBOMEntry.Metadata).To(Equal(paketosbom.BOMMetadata{
			Version: "dotnet-sdk-dep-version",
			Checksum: paketosbom.BOMChecksum{
				Algorithm: paketosbom.SHA256,
				Hash:      "dotnet-sdk-dep-sha",
			},
			URI: "dotnet-sdk-dep-uri",
		}))

		Expect(entryResolver.ResolveCall.Receives.Entries).
			To(Equal([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-sdk",
					Metadata: map[string]interface{}{
						"version-source": "some-source",
						"version":        "2.5.x",
						"build":          true,
						"launch":         true,
					},
				},
			}))
		Expect(entryResolver.MergeLayerTypesCall.Receives.Entries).To(Equal([]packit.BuildpackPlanEntry{
			{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version-source": "some-source",
					"version":        "2.5.x",
					"build":          true,
					"launch":         true,
				},
			},
		}))

		Expect(dependencyManager.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(dependencyManager.ResolveCall.Receives.Id).To(Equal("dotnet-sdk"))
		Expect(dependencyManager.ResolveCall.Receives.Version).To(Equal("2.5.x"))
		Expect(dependencyManager.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
			{
				ID:      "dotnet-sdk",
				Version: "some-version",
				Name:    ".NET Core SDK",
				SHA256:  "some-sha",
			},
		}))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).
			To(Equal(postal.Dependency{
				ID:      "dotnet-sdk",
				Name:    ".NET Core SDK",
				Version: "some-version",
				SHA256:  "some-sha",
			}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-sdk")))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "dotnet-sdk",
			Name:    ".NET Core SDK",
			Version: "some-version",
			SHA256:  "some-sha",
		}))
		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dir).To(Equal(filepath.Join(layersDir, "dotnet-core-sdk")))
	})

	context("when there is a dependency cache match", func() {
		it.Before(func() {
			err := os.WriteFile(filepath.Join(layersDir, "dotnet-core-sdk.toml"),
				[]byte("[metadata]\ndependency-sha = \"some-sha\"\n"), 0600)
			Expect(err).NotTo(HaveOccurred())

			Expect(os.MkdirAll(filepath.Join(layersDir, "dotnet-core-sdk"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layersDir, "dotnet-core-sdk", "dotnet"), []byte(`hi`), os.ModePerm)).To(Succeed())

			entryResolver.MergeLayerTypesCall.Returns.Build = true
			entryResolver.MergeLayerTypesCall.Returns.Launch = false
		})

		it("reuses the cached version of the SDK dependency", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Version: "1.2.3",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "dotnet-sdk",
						},
					},
				},
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				WorkingDir: workingDir,
				Stack:      "some-stack",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(2))
			envLayer := result.Layers[0]
			SDKLayer := result.Layers[1]

			Expect(SDKLayer.Name).To(Equal("dotnet-core-sdk"))
			Expect(SDKLayer.BuildEnv).To(Equal(packit.Environment{
				"PATH.prepend": filepath.Join(layersDir, "dotnet-core-sdk"),
				"PATH.delim":   string(os.PathListSeparator),
			}))
			Expect(envLayer.LaunchEnv).To(Equal(packit.Environment{
				"PATH.prepend": filepath.Join(workingDir, ".dotnet_root"),
				"PATH.delim":   string(os.PathListSeparator),
			}))
			Expect(SDKLayer.Path).To(Equal(filepath.Join(layersDir, "dotnet-core-sdk")))
			Expect(SDKLayer.Metadata).To(Equal(map[string]interface{}{
				"dependency-sha": "some-sha",
			}))

			Expect(SDKLayer.Build).To(BeTrue())
			Expect(SDKLayer.Launch).To(BeFalse())
			Expect(SDKLayer.Cache).To(BeTrue())

			Expect(envLayer.Build).To(BeFalse())
			Expect(envLayer.Launch).To(BeTrue())
			Expect(envLayer.Cache).To(BeFalse())

			Expect(result.Build.BOM).To(HaveLen(1))
			buildBOMEntry := result.Build.BOM[0]
			Expect(buildBOMEntry.Name).To(Equal("dotnet-sdk"))
			Expect(buildBOMEntry.Metadata).To(Equal(paketosbom.BOMMetadata{
				Version: "dotnet-sdk-dep-version",
				Checksum: paketosbom.BOMChecksum{
					Algorithm: paketosbom.SHA256,
					Hash:      "dotnet-sdk-dep-sha",
				},
				URI: "dotnet-sdk-dep-uri",
			}))

			Expect(dependencyManager.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
			Expect(dependencyManager.ResolveCall.Receives.Id).To(Equal("dotnet-sdk"))
			Expect(dependencyManager.ResolveCall.Receives.Version).To(Equal("2.5.x"))
			Expect(dependencyManager.ResolveCall.Receives.Stack).To(Equal("some-stack"))

			Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
				{
					ID:      "dotnet-sdk",
					Version: "some-version",
					Name:    ".NET Core SDK",
					SHA256:  "some-sha",
				},
			}))

			Expect(dependencyManager.DeliverCall.CallCount).To(Equal(0))
		})
	})

	context("when the sdk version is set via buildpack.yml", func() {
		it.Before(func() {
			entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
				Name: "dotnet-sdk",
				Metadata: map[string]interface{}{
					"version":        "2.5.x",
					"version-source": "buildpack.yml",
					"build":          true,
					"launch":         true,
				},
			}
		})

		it("logs a deprecation warning", func() {
			_, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Version: "1.2.3",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "dotnet-sdk",
						},
					},
				},
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				WorkingDir: workingDir,
				Stack:      "some-stack",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(buffer.String()).To(ContainSubstring("WARNING: Setting the .NET Core SDK version through buildpack.yml will be deprecated soon in .NET Core SDK Buildpack v2.0.0"))
		})
	})

	context("failure cases", func() {
		context("when the dependency for the build plan entry cannot be resolved", func() {
			it.Before(func() {
				dependencyManager.ResolveCall.Returns.Error = errors.New("some-resolution-error")
			})
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-sdk",
							},
						},
					},
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					WorkingDir: workingDir,
					Stack:      "some-stack",
				})

				Expect(err).To(MatchError("some-resolution-error"))
			})
		})

		context("when layer dir cannot be accessed", func() {
			it.Before(func() {
				Expect(os.Chmod(layersDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, 0600)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{
						Version: "1.2.3",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-sdk",
							},
						},
					},
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					WorkingDir: workingDir,
					Stack:      "some-stack",
				})

				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the dotnet CLI cannot be copied into the workspace/.dotnet_root", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, ".dotnet_root"), 0000)).To(Succeed())
			})
			it.After(func() {
				Expect(os.Chmod(filepath.Join(workingDir, ".dotnet_root"), os.ModePerm)).To(Succeed())
			})
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{
						Version: "1.2.3",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-sdk",
							},
						},
					},
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					WorkingDir: workingDir,
					Stack:      "some-stack",
				})

				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the workspace/.dotnet_root directory cannot be made", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
			})
			it.After(func() {
				Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
			})
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{
						Version: "1.2.3",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-sdk",
							},
						},
					},
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					WorkingDir: workingDir,
					Stack:      "some-stack",
				})

				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when layer cannot be removed", func() {
			var layerDir string
			it.Before(func() {
				layerDir = filepath.Join(layersDir, "dotnet-core-sdk")
				Expect(os.MkdirAll(filepath.Join(layerDir, "dotnet-core-sdk"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(layerDir, 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layerDir, os.ModePerm)).To(Succeed())
				Expect(os.RemoveAll(layerDir)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{
						Version: "1.2.3",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-sdk",
							},
						},
					},
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					WorkingDir: workingDir,
					Stack:      "some-stack",
				})

				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the dependency for the build plan entry cannot be resolved", func() {
			it.Before(func() {
				dependencyManager.DeliverCall.Stub = func(postal.Dependency, string, string, string) error {
					return errors.New("some-installation-error")
				}
			})
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{
						Version: "1.2.3",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-sdk",
							},
						},
					},
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					WorkingDir: workingDir,
					Stack:      "some-stack",
				})

				Expect(err).To(MatchError("some-installation-error"))
			})
		})

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateFromDependencyCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{
						Version: "1.2.3",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-sdk",
							},
						},
					},
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					WorkingDir: workingDir,
					Stack:      "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{SBOMFormats: []string{"random-format"}},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-sdk",
							},
						},
					},
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					WorkingDir: workingDir,
					Stack:      "some-stack",
				})
				Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
			})
		})
	})
}
