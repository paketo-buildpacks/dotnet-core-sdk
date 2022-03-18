package dotnetcoresdk_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/dotnet-core-sdk/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"

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
		clock      chronos.Clock
		timeStamp  time.Time
		buffer     *bytes.Buffer

		entryResolver     *fakes.EntryResolver
		dependencyMapper  *fakes.DependencyMapper
		dependencyManager *fakes.DependencyManager
		dotnetSymlinker   *fakes.DotnetSymlinker

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error

		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		dependencyMapper = &fakes.DependencyMapper{}
		dependencyMapper.FindCorrespondingVersionCall.Returns.String = "1.2.300"

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
			Name:    "Dotnet Core SDK",
			SHA256:  "some-sha",
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

		dotnetSymlinker = &fakes.DotnetSymlinker{}

		buffer = bytes.NewBuffer(nil)
		logEmitter := dotnetcoresdk.NewLogEmitter(buffer)

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		build = dotnetcoresdk.Build(
			entryResolver,
			dependencyMapper,
			dependencyManager,
			dotnetSymlinker,
			logEmitter,
			clock,
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
				Version: "1.2.3",
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
		Expect(result).To(Equal(packit.BuildResult{
			Layers: []packit.Layer{
				{
					Name:             "dotnet-core-sdk",
					Path:             filepath.Join(layersDir, "dotnet-core-sdk"),
					SharedEnv:        packit.Environment{},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            true,
					Launch:           true,
					Cache:            true,
					Metadata: map[string]interface{}{
						"dependency-sha": "some-sha",
						"built_at":       timeStamp.Format(time.RFC3339Nano),
					},
				},
				{
					Name: "dotnet-env-var",
					Path: filepath.Join(layersDir, "dotnet-env-var"),
					SharedEnv: packit.Environment{
						"PATH.prepend":         filepath.Join(workingDir, ".dotnet_root"),
						"PATH.delim":           string(os.PathListSeparator),
						"DOTNET_ROOT.override": filepath.Join(workingDir, ".dotnet_root"),
					},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            true,
					Launch:           true,
				},
			},
			Build: packit.BuildMetadata{
				BOM: []packit.BOMEntry{
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
				},
			},
			Launch: packit.LaunchMetadata{
				BOM: []packit.BOMEntry{
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
				},
			},
		}))

		Expect(dependencyMapper.FindCorrespondingVersionCall.CallCount).To(Equal(0))
		Expect(os.Getenv("RUNTIME_VERSION")).To(Equal(""))

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
				Name:    "Dotnet Core SDK",
				SHA256:  "some-sha",
			},
		}))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).
			To(Equal(postal.Dependency{
				ID:      "dotnet-sdk",
				Name:    "Dotnet Core SDK",
				Version: "some-version",
				SHA256:  "some-sha",
			}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-sdk")))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(dotnetSymlinker.LinkCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(dotnetSymlinker.LinkCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-sdk")))
	})

	context("when RUNTIME_VERSION is set", func() {
		it.Before(func() {
			os.Setenv("RUNTIME_VERSION", "1.2.3")
		})

		it.After(func() {
			os.Unsetenv("RUNTIME_VERSION")
		})
		it("uses the dependency mapper and adds an entry to the build plan", func() {
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

			Expect(dependencyMapper.FindCorrespondingVersionCall.CallCount).To(Equal(1))
			Expect(dependencyMapper.FindCorrespondingVersionCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
			Expect(dependencyMapper.FindCorrespondingVersionCall.Receives.VersionKey).To(Equal("1.2.3"))

			Expect(entryResolver.ResolveCall.Receives.Entries).
				To(Equal([]packit.BuildpackPlanEntry{
					{
						Name: "dotnet-sdk",
					},
					{
						Name: "dotnet-sdk",
						Metadata: map[string]interface{}{
							"version-source": "RUNTIME_VERSION",
							"version":        "1.2.300",
						},
					},
				}))
		})
		context("when looking for a compatible SDK version fails", func() {
			it.Before(func() {
				dependencyMapper.FindCorrespondingVersionCall.Returns.Error = errors.New("some-mapping-error")
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

				Expect(err).To(MatchError("some-mapping-error"))
			})
		})
	})

	context("when there is a dependency cache match", func() {
		it.Before(func() {
			err := os.WriteFile(filepath.Join(layersDir, "dotnet-core-sdk.toml"),
				[]byte("[metadata]\ndependency-sha = \"some-sha\"\n"), 0600)
			Expect(err).NotTo(HaveOccurred())

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
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:             "dotnet-core-sdk",
						Path:             filepath.Join(layersDir, "dotnet-core-sdk"),
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           false,
						Cache:            true,
						Metadata: map[string]interface{}{
							"dependency-sha": "some-sha",
						},
					},
					{
						Name: "dotnet-env-var",
						Path: filepath.Join(layersDir, "dotnet-env-var"),
						SharedEnv: packit.Environment{
							"PATH.prepend":         filepath.Join(workingDir, ".dotnet_root"),
							"PATH.delim":           string(os.PathListSeparator),
							"DOTNET_ROOT.override": filepath.Join(workingDir, ".dotnet_root"),
						},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           true,
					},
				},
				Build: packit.BuildMetadata{
					BOM: []packit.BOMEntry{
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
					Name:    "Dotnet Core SDK",
					SHA256:  "some-sha",
				},
			}))

			Expect(dependencyManager.DeliverCall.CallCount).To(Equal(0))

			Expect(dotnetSymlinker.LinkCall.CallCount).To(Equal(1))
			Expect(dotnetSymlinker.LinkCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(dotnetSymlinker.LinkCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-sdk")))
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

			Expect(buffer.String()).To(ContainSubstring("WARNING: Setting the .NET Core SDK version through buildpack.yml will be deprecated soon in Dotnet Core SDK Buildpack v2.0.0"))
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
				dependencyManager.DeliverCall.Returns.Error = errors.New("some-installation-error")
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

		context("when symlinking fails", func() {
			it.Before(func() {
				dotnetSymlinker.LinkCall.Returns.Error = errors.New("some-symlinking-error")
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

				Expect(err).To(MatchError("some-symlinking-error"))
			})
		})
	})
}
