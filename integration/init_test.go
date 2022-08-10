package integration_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var settings struct {
	BuildpackInfo struct {
		Buildpack struct {
			ID   string
			Name string
		}
		Metadata struct {
			Dependencies []postal.Dependency `toml:"dependencies"`
		} `toml:"metadata"`
	}

	Config struct {
		BuildPlan string `json:"build-plan"`
	}

	Buildpacks struct {
		DotnetCoreSDK struct {
			Online  string
			Offline string
		}
		BuildPlan struct {
			Online string
		}
	}
}

var builder struct {
	Local struct {
		Stack struct {
			ID string `json:"id"`
		} `json:"stack"`
	} `json:"local_info"`
}

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect

	root, err := filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	file, err := os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.NewDecoder(file).Decode(&settings.BuildpackInfo)
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())

	file, err = os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())

	Expect(json.NewDecoder(file).Decode(&settings.Config)).To(Succeed())
	Expect(file.Close()).To(Succeed())

	buildpackStore := occam.NewBuildpackStore()

	settings.Buildpacks.DotnetCoreSDK.Online, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.DotnetCoreSDK.Offline, err = buildpackStore.Get.
		WithOfflineDependencies().
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.BuildPlan.Online, err = buildpackStore.Get.
		Execute(settings.Config.BuildPlan)
	Expect(err).NotTo(HaveOccurred())

	SetDefaultEventuallyTimeout(5 * time.Second)

	buf := bytes.NewBuffer(nil)
	cmd := pexec.NewExecutable("pack")
	Expect(cmd.Execute(pexec.Execution{
		Args:   []string{"builder", "inspect", "--output", "json"},
		Stdout: buf,
		Stderr: buf,
	})).To(Succeed(), buf.String())

	Expect(json.Unmarshal(buf.Bytes(), &builder)).To(Succeed(), buf.String())

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Default", testDefault)
	suite("LayerReuse", testLayerReuse)
	suite("Offline", testOffline)
	suite.Run(t)
}
