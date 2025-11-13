package components

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/libdependency/retrieve"
	"github.com/paketo-buildpacks/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
)

func GenerateMetadata(version versionology.VersionFetcher, platform retrieve.Platform) ([]versionology.Dependency, error) {
	sdkRelease := version.(SdkRelease)

	arch := platform.Arch
	if platform.Arch == "amd64" {
		arch = "x64"
	}
	rid := fmt.Sprintf("%s-%s", platform.OS, arch)

	var archive ReleaseFile
	for _, file := range sdkRelease.Files {
		if file.Rid == rid && strings.HasSuffix(file.Name, ".tar.gz") {
			archive = file
			break
		}
	}

	if (archive == ReleaseFile{}) {
		return nil, fmt.Errorf("could not find release file for %s/%s", platform.OS, arch)
	}

	// Validate the artifact
	response, err := http.Get(archive.URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	vr := cargo.NewValidatedReader(response.Body, fmt.Sprintf("sha512:%s", archive.Hash))
	valid, err := vr.Valid()
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, fmt.Errorf("the given checksum of the artifact does not match with downloaded artifact")
	}

	var depDate *time.Time
	if sdkRelease.EOLDate != "" {
		t, err := time.ParseInLocation("2006-01-02", sdkRelease.EOLDate, time.UTC)
		if err != nil {
			return nil, err
		}
		depDate = &t
	}

	productName := ".net"
	if sdkRelease.SemVer.LessThan(semver.MustParse("5.0.0-0")) { // use 5.0.0-0 to ensure 5.0.0 previews/RCs use the new `.net` product name
		productName = ".net_core"
	}

	cpe := fmt.Sprintf("cpe:2.3:a:microsoft:%s:%s:*:*:*:*:*:*:*", productName, sdkRelease.ReleaseVersion)
	purl := retrieve.GeneratePURL("dotnet-core-sdk", sdkRelease.ReleaseVersion, archive.Hash, archive.URL)

	metadataDependency := cargo.ConfigMetadataDependency{
		ID:              "dotnet-sdk",
		Name:            ".NET Core SDK",
		Version:         sdkRelease.SemVer.String(),
		Stacks:          []string{"*"},
		DeprecationDate: depDate,
		URI:             archive.URL,
		Checksum:        fmt.Sprintf("sha512:%s", archive.Hash),
		Source:          archive.URL,
		SourceChecksum:  fmt.Sprintf("sha512:%s", archive.Hash),
		CPE:             cpe,
		PURL:            purl,
		Licenses:        []interface{}{"MIT", "MIT-0"},
		OS:              platform.OS,
		Arch:            platform.Arch,
	}

	dependency, err := versionology.NewDependency(metadataDependency, "*")
	if err != nil {
		return nil, err
	}

	return []versionology.Dependency{dependency}, nil
}
