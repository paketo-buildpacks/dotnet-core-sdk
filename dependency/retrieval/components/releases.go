package components

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Masterminds/semver/v3"
)

type Release struct {
	SemVer *semver.Version

	EOLDate string

	Version string        `json:"version"`
	Files   []ReleaseFile `json:"files"`
}

type ReleaseFile struct {
	Name string `json:"name"`
	Rid  string `json:"rid"`
	URL  string `json:"url"`
	Hash string `json:"hash"`
}

type Fetcher struct {
	releaseIndex string
}

func NewFetcher() Fetcher {
	return Fetcher{
		releaseIndex: "https://dotnetcli.blob.core.windows.net/dotnet/release-metadata/releases-index.json",
	}
}

func (f Fetcher) WithReleaseIndex(uri string) Fetcher {
	f.releaseIndex = uri
	return f
}

func (f Fetcher) Get() ([]Release, error) {
	response, err := http.Get(f.releaseIndex)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if !(response.StatusCode >= 200 && response.StatusCode < 300) {
		return nil, fmt.Errorf("received a non 200 status code from %s: status code %d received", f.releaseIndex, response.StatusCode)
	}

	var releasesIndex struct {
		ReleasesIndex []struct {
			ReleaseJSON string `json:"releases.json"`
		} `json:"releases-index"`
	}

	err = json.NewDecoder(response.Body).Decode(&releasesIndex)
	if err != nil {
		return nil, err
	}

	var releases []Release
	for _, releaseIndex := range releasesIndex.ReleasesIndex {
		releaseResponse, err := http.Get(releaseIndex.ReleaseJSON)
		if err != nil {
			return nil, err
		}
		defer releaseResponse.Body.Close()

		if !(releaseResponse.StatusCode >= 200 && releaseResponse.StatusCode < 300) {
			return nil, fmt.Errorf("received a non 200 status code from %s: status code %d received", releaseIndex.ReleaseJSON, releaseResponse.StatusCode)
		}

		var releasePage struct {
			EOLDate  string `json:"eol-date"`
			Releases []struct {
				Release Release `json:"sdk"`
			} `json:"releases"`
		}

		err = json.NewDecoder(releaseResponse.Body).Decode(&releasePage)
		if err != nil {
			return nil, err
		}

		for _, r := range releasePage.Releases {
			release := r.Release

			release.EOLDate = releasePage.EOLDate
			release.SemVer, err = semver.NewVersion(release.Version)
			if err != nil {
				return nil, err
			}

			releases = append(releases, release)
		}

	}

	return releases, nil
}
