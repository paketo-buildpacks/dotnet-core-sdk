package dotnetcoresdk_test

import (
	"os"
	"path/filepath"
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSymlinker(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		symlinker  dotnetcoresdk.Symlinker
		workingDir string
		layerPath  string
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		layerPath, err = os.MkdirTemp("", "layer-path")
		Expect(err).NotTo(HaveOccurred())

		symlinker = dotnetcoresdk.NewSymlinker()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(layerPath)).To(Succeed())
	})

	context("Link", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(layerPath, "dotnet"), []byte{}, os.ModePerm)).To(Succeed())
		})

		it("creates a .dotnet_root dir in workspace with symlink to layerpath", func() {
			err := symlinker.Link(workingDir, layerPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(workingDir, ".dotnet_root")).To(BeADirectory())

			fi, err := os.Lstat(filepath.Join(workingDir, ".dotnet_root", "sdk"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fi.Mode() & os.ModeSymlink).ToNot(BeZero())

			link, err := os.Readlink(filepath.Join(workingDir, ".dotnet_root", "sdk"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "sdk")))

			fi, err = os.Lstat(filepath.Join(workingDir, ".dotnet_root", "templates"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fi.Mode() & os.ModeSymlink).ToNot(BeZero())

			link, err = os.Readlink(filepath.Join(workingDir, ".dotnet_root", "templates"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "templates")))

			fi, err = os.Lstat(filepath.Join(workingDir, ".dotnet_root", "packs"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fi.Mode() & os.ModeSymlink).ToNot(BeZero())

			link, err = os.Readlink(filepath.Join(workingDir, ".dotnet_root", "packs"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "packs")))

			Expect(filepath.Join(workingDir, ".dotnet_root", "dotnet")).To(BeAnExistingFile())
		})

		context("error cases", func() {
			context("when the '.dotnet_root/sdk' dir can not be created", func() {
				it.Before(func() {
					Expect(os.Chmod(filepath.Join(workingDir), 0000)).To(Succeed())
				})
				it("errors", func() {
					err := symlinker.Link(workingDir, layerPath)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the shared directory symlink can not be created", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(workingDir, ".dotnet_root"), os.ModePerm)).To(Succeed())
					Expect(os.Chmod(filepath.Join(workingDir, ".dotnet_root"), 0000)).To(Succeed())
				})
				it("errors", func() {
					err := symlinker.Link(workingDir, layerPath)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when dotnet executable cannot be copied", func() {
				it.Before(func() {
					Expect(os.RemoveAll(filepath.Join(layerPath, "dotnet"))).To(Succeed())
				})

				it("errors", func() {
					err := symlinker.Link(workingDir, layerPath)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})
		})
	})
}
