package components

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/packit/v2/cargo"
)

func ConvertReleaseToDependency(release Release) (cargo.ConfigMetadataDependency, error) {
	var archive ReleaseFile
	for _, file := range release.Files {
		if file.Rid == "linux-x64" && strings.HasSuffix(file.Name, ".tar.gz") {
			archive = file
			break
		}
	}

	if (archive == ReleaseFile{}) {
		return cargo.ConfigMetadataDependency{}, fmt.Errorf("could not find release file for linux/x64")
	}

	purl := GeneratePURL("dotnet-core-sdk", release.Version, archive.Hash, archive.URL)

	licenses, err := GenerateLicenseInformation(archive.URL)
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}

	// Validate the artifact
	response, err := http.Get(archive.URL)
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}
	defer response.Body.Close()

	vr := cargo.NewValidatedReader(response.Body, fmt.Sprintf("sha512:%s", archive.Hash))
	valid, err := vr.Valid()
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}

	if !valid {
		return cargo.ConfigMetadataDependency{}, fmt.Errorf("the given checksum of the artifact does not match with downloaded artifact")
	}

	stacks := []string{
		"io.buildpacks.stacks.bionic",
	}

	c, _ := semver.NewConstraint("3.1.*")
	if !(c.Check(release.SemVer)) {
		stacks = append(stacks, "io.buildpacks.stacks.jammy")
	}

	var depDate *time.Time
	if release.EOLDate != "" {
		t, err := time.ParseInLocation("2006-01-02", release.EOLDate, time.UTC)
		if err != nil {
			return cargo.ConfigMetadataDependency{}, err
		}
		depDate = &t
	}

	productName := ".net"
	if release.SemVer.LessThan(semver.MustParse("5.0.0-0")) { // use 5.0.0-0 to ensure 5.0.0 previews/RCs use the new `.net` product name
		productName = ".net_core"
	}

	return cargo.ConfigMetadataDependency{
		ID:              "dotnet-sdk",
		Name:            ".NET Core SDK",
		Version:         release.SemVer.String(),
		Stacks:          stacks,
		DeprecationDate: depDate,
		URI:             archive.URL,
		Checksum:        fmt.Sprintf("sha512:%s", archive.Hash),
		Source:          archive.URL,
		SourceChecksum:  fmt.Sprintf("sha512:%s", archive.Hash),
		CPE:             fmt.Sprintf("cpe:2.3:a:microsoft:%s:%s:*:*:*:*:*:*:*", productName, release.Version),
		PURL:            purl,
		Licenses:        licenses,
	}, nil
}
