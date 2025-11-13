package components_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/dotnet-core-sdk/dependency/retrieval/components"
	"github.com/paketo-buildpacks/libdependency/versionology"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testReleases(t *testing.T, context spec.G, it spec.S) {

	var (
		Expect = NewWithT(t).Expect
	)

	context("GetReleases", func() {
		var (
			fetcher components.Fetcher

			releaseIndex *httptest.Server
			releasePage  *httptest.Server
		)

		it.Before(func() {
			releasePage = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodHead {
					http.Error(w, "NotFound", http.StatusNotFound)
					return
				}

				switch req.URL.Path {
				case "/6.0":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{
	"eol-date": "2024-11-12",
	"releases": [{
		"sdk": {
			"version": "6.0.401",
			"files": [{
				"name": "dotnet-sdk-linux-arm.tar.gz",
				"rid": "linux-arm",
				"url": "https://download.visualstudio.microsoft.com/download/pr/451f282f-dd26-4acd-9395-36cc3a9758e4/f5399d2ebced2ad9640db6283aa9d714/dotnet-sdk-6.0.401-linux-arm.tar.gz",
				"hash": "7d3c32f510a7298b8e4c32a95e7d3c9b0475d94510732a405163c7bff589ffda8964f2e9336d560bd1dc37461e6cb3da5809337a586da0288bdcc71496013ba0"
			}]
		}
	},
  {
		"sdk": {
			"version": "6.0.400",
			"files": [{
				"name": "dotnet-sdk-linux-arm.tar.gz",
        "rid": "linux-arm",
        "url": "https://download.visualstudio.microsoft.com/download/pr/5a24144e-0d7d-4cc9-b9d8-b4d32d6bb084/e882181e475e3c66f48a22fbfc7b19c0/dotnet-sdk-6.0.400-linux-arm.tar.gz",
        "hash": "a72aa70bfb15e21a20ddd90c2c3e37acb53e6f1e50f5b6948aac616b28f80ac81e1157e8db5688e21dc9a7496011ef0fcf06cdca74ddc7271f9a1c6268f4b1b2"
			}]
		}
	}
	]
}`)
				case "/3.1":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{
	"eol-date": "2022-12-13",
	"releases": [{
		"sdk": {
			"version": "3.1.423",
			"files": [{
				"name": "dotnet-sdk-linux-arm.tar.gz",
				"rid": "linux-arm",
				"url": "https://download.visualstudio.microsoft.com/download/pr/8f81b133-220b-4831-abe6-e8be161fd9a2/1af75b5e2ca89af2a31cf9981a976832/dotnet-sdk-3.1.423-linux-arm.tar.gz",
				"hash": "6b615ec6c1d66280c44ff28de0532ff6a4c21c77caf188101b04bdd58e8935436cb2b049ad9d831799476d421e25795184615c7e1caff8e550855e2f6ed5efd9"
			}]
		}
	}
	]
}`)

				case "/non-200":
					w.WriteHeader(http.StatusTeapot)

				case "/no-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `???`)

				case "/no-version-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{
	"eol-date": "2022-12-13",
	"releases": [{
		"sdk": {
			"version": "invalid version",
			"files": [{
				"name": "dotnet-sdk-linux-arm.tar.gz",
				"rid": "linux-arm",
				"url": "https://download.visualstudio.microsoft.com/download/pr/8f81b133-220b-4831-abe6-e8be161fd9a2/1af75b5e2ca89af2a31cf9981a976832/dotnet-sdk-3.1.423-linux-arm.tar.gz",
				"hash": "6b615ec6c1d66280c44ff28de0532ff6a4c21c77caf188101b04bdd58e8935436cb2b049ad9d831799476d421e25795184615c7e1caff8e550855e2f6ed5efd9"
			}]
		}
	}
	]
}`)

				default:
					t.Fatalf("unknown path: %s", req.URL.Path)
				}
			}))

			releaseIndex = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodHead {
					http.Error(w, "NotFound", http.StatusNotFound)
					return
				}

				switch req.URL.Path {
				case "/":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, `{
    "releases-index": [
        {
            "releases.json": "%[1]s/6.0"
        },
				{
            "releases.json": "%[1]s/3.1"
				}
    ]
}\n`, releasePage.URL)

				case "/index-non-200":
					w.WriteHeader(http.StatusTeapot)

				case "/index-no-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `???`)

				case "/release-get-fails":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{
    "releases-index": [
				{
            "releases.json": "Not a valid URL"
				}
    ]
}`)

				case "/release-non-200":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, `{
    "releases-index": [
				{
            "releases.json": "%s/non-200"
				}
    ]
}\n`, releasePage.URL)

				case "/release-no-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, `{
    "releases-index": [
				{
            "releases.json": "%s/no-parse"
				}
    ]
}\n`, releasePage.URL)

				case "/release-no-version-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, `{
    "releases-index": [
				{
            "releases.json": "%s/no-version-parse"
				}
    ]
}\n`, releasePage.URL)

				default:
					t.Fatalf("unknown path: %s", req.URL.Path)
				}
			}))

			fetcher = components.NewFetcher().WithReleaseIndex(releaseIndex.URL)
		})

		it("fetches a list of relevant releases", func() {
			releases, err := fetcher.GetVersions()
			Expect(err).NotTo(HaveOccurred())

			Expect(releases).To(BeEquivalentTo([]versionology.VersionFetcher{
				components.SdkRelease{
					SemVer:         semver.MustParse("6.0.401"),
					EOLDate:        "2024-11-12",
					ReleaseVersion: "6.0.401",
					Files: []components.ReleaseFile{
						{
							Name: "dotnet-sdk-linux-arm.tar.gz",
							Rid:  "linux-arm",
							URL:  "https://download.visualstudio.microsoft.com/download/pr/451f282f-dd26-4acd-9395-36cc3a9758e4/f5399d2ebced2ad9640db6283aa9d714/dotnet-sdk-6.0.401-linux-arm.tar.gz",
							Hash: "7d3c32f510a7298b8e4c32a95e7d3c9b0475d94510732a405163c7bff589ffda8964f2e9336d560bd1dc37461e6cb3da5809337a586da0288bdcc71496013ba0",
						},
					},
				},
				components.SdkRelease{
					SemVer:         semver.MustParse("6.0.400"),
					EOLDate:        "2024-11-12",
					ReleaseVersion: "6.0.400",
					Files: []components.ReleaseFile{
						{
							Name: "dotnet-sdk-linux-arm.tar.gz",
							Rid:  "linux-arm",
							URL:  "https://download.visualstudio.microsoft.com/download/pr/5a24144e-0d7d-4cc9-b9d8-b4d32d6bb084/e882181e475e3c66f48a22fbfc7b19c0/dotnet-sdk-6.0.400-linux-arm.tar.gz",
							Hash: "a72aa70bfb15e21a20ddd90c2c3e37acb53e6f1e50f5b6948aac616b28f80ac81e1157e8db5688e21dc9a7496011ef0fcf06cdca74ddc7271f9a1c6268f4b1b2",
						},
					},
				},
				components.SdkRelease{
					SemVer:         semver.MustParse("3.1.423"),
					EOLDate:        "2022-12-13",
					ReleaseVersion: "3.1.423",
					Files: []components.ReleaseFile{
						{
							Name: "dotnet-sdk-linux-arm.tar.gz",
							Rid:  "linux-arm",
							URL:  "https://download.visualstudio.microsoft.com/download/pr/8f81b133-220b-4831-abe6-e8be161fd9a2/1af75b5e2ca89af2a31cf9981a976832/dotnet-sdk-3.1.423-linux-arm.tar.gz",
							Hash: "6b615ec6c1d66280c44ff28de0532ff6a4c21c77caf188101b04bdd58e8935436cb2b049ad9d831799476d421e25795184615c7e1caff8e550855e2f6ed5efd9",
						},
					},
				},
			}))
		})

		context("failure cases", func() {
			context("when the index page get fails", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex("not a valid URL")
				})

				it("returns an error", func() {
					_, err := fetcher.GetVersions()
					Expect(err).To(MatchError(ContainSubstring("unsupported protocol scheme")))
				})
			})

			context("when the index page returns non 200 code", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/index-non-200", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.GetVersions()
					Expect(err).To(MatchError(fmt.Sprintf("received a non 200 status code from %s: status code 418 received", fmt.Sprintf("%s/index-non-200", releaseIndex.URL))))
				})
			})

			context("when the index page cannot be parsed", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/index-no-parse", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.GetVersions()
					Expect(err).To(MatchError(ContainSubstring("invalid character '?' looking for beginning of value")))
				})
			})

			context("when the release page get fails", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/release-get-fails", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.GetVersions()
					Expect(err).To(MatchError(ContainSubstring("unsupported protocol scheme")))
				})
			})

			context("when the release page non 200 code", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/release-non-200", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.GetVersions()
					Expect(err).To(MatchError(fmt.Sprintf("received a non 200 status code from %s: status code 418 received", fmt.Sprintf("%s/non-200", releasePage.URL))))
				})
			})

			context("when the release page cannot be parsed", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/release-no-parse", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.GetVersions()
					Expect(err).To(MatchError(ContainSubstring("invalid character '?' looking for beginning of value")))
				})
			})

			context("when the release page has unparsable version", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/release-no-version-parse", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.GetVersions()
					Expect(err).To(MatchError(ContainSubstring("invalid semantic version")))
				})
			})
		})
	})
}
