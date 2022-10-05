package components_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/dotnet-core-sdk/dependency/retrieval/components"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

const (
	lFile = `The MIT License (MIT)

Copyright (c) .NET Foundation and Contributors

All rights reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`
)

func testDependency(t *testing.T, context spec.G, it spec.S) {

	var (
		Expect = NewWithT(t).Expect
	)

	context("ConvertReleaseToDependeny", func() {
		var (
			server *httptest.Server
		)

		it.Before(func() {
			buffer := bytes.NewBuffer(nil)
			gw := gzip.NewWriter(buffer)
			tw := tar.NewWriter(gw)

			licenseFile := "./LICENSE.txt"
			Expect(tw.WriteHeader(&tar.Header{Name: licenseFile, Mode: 0755, Size: int64(len(lFile))})).To(Succeed())
			_, err := tw.Write([]byte(lFile))
			Expect(err).NotTo(HaveOccurred())

			Expect(tw.Close()).To(Succeed())
			Expect(gw.Close()).To(Succeed())

			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodHead {
					http.Error(w, "NotFound", http.StatusNotFound)
					return
				}

				switch req.URL.Path {
				case "/":
					w.WriteHeader(http.StatusOK)
					_, err := w.Write(buffer.Bytes())
					Expect(err).NotTo(HaveOccurred())

				case "/no-license":
					w.WriteHeader(http.StatusOK)
					buffer = bytes.NewBuffer(nil)
					gw = gzip.NewWriter(buffer)
					tw = tar.NewWriter(gw)

					licenseFile = "./NO-LICENSE.txt"
					Expect(tw.WriteHeader(&tar.Header{Name: licenseFile, Mode: 0755, Size: int64(len(`some-content`))})).To(Succeed())
					_, err = tw.Write([]byte(`some-content`))
					Expect(err).NotTo(HaveOccurred())

					Expect(tw.Close()).To(Succeed())
					Expect(gw.Close()).To(Succeed())

					_, err = w.Write(buffer.Bytes())
					Expect(err).NotTo(HaveOccurred())

				default:
					t.Fatalf("unknown path: %s", req.URL.Path)
				}
			}))

		})

		it("returns returns a cargo dependency generated from the given release", func() {
			dependency, err := components.ConvertReleaseToDependency(components.Release{
				SemVer:  semver.MustParse("6.0.401"),
				EOLDate: "2024-11-12",
				Version: "6.0.401",
				Files: []components.ReleaseFile{
					{
						Name: "dotnet-sdk-linux-x64.tar.gz",
						Rid:  "linux-x64",
						URL:  server.URL,
						Hash: "365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6",
					},
				},
			},
			)
			Expect(err).NotTo(HaveOccurred())

			depDate, err := time.ParseInLocation("2006-01-02", "2024-11-12", time.UTC)
			Expect(err).NotTo(HaveOccurred())

			Expect(dependency).To(Equal(cargo.ConfigMetadataDependency{
				Checksum:        "sha512:365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6",
				CPE:             "cpe:2.3:a:microsoft:.net:6.0.401:*:*:*:*:*:*:*",
				PURL:            fmt.Sprintf("pkg:generic/dotnet-core-sdk@6.0.401?checksum=365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6&download_url=%s", server.URL),
				DeprecationDate: &depDate,
				ID:              "dotnet-core-sdk",
				Licenses:        []interface{}{"MIT", "MIT-0"},
				Name:            ".NET Core SDK",
				SHA256:          "",
				Source:          server.URL,
				SourceChecksum:  "sha512:365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6",
				SourceSHA256:    "",
				Stacks: []string{
					"io.buildpacks.stacks.bionic",
					"io.paketo.stacks.tiny",
					"io.buildpacks.stacks.jammy",
					"io.buildpacks.stacks.jammy.tiny",
				},
				StripComponents: 0,
				URI:             server.URL,
				Version:         "6.0.401",
			}))
		})

		context("when the release is 3.1.*", func() {
			it("returns returns a cargo dependency generated from the given release with different purl and stacks", func() {
				dependency, err := components.ConvertReleaseToDependency(components.Release{
					SemVer:  semver.MustParse("3.1.423"),
					EOLDate: "2022-12-13",
					Version: "3.1.423",
					Files: []components.ReleaseFile{
						{
							Name: "dotnet-sdk-linux-x64.tar.gz",
							Rid:  "linux-x64",
							URL:  server.URL,
							Hash: "365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6",
						},
					},
				},
				)
				Expect(err).NotTo(HaveOccurred())

				depDate, err := time.ParseInLocation("2006-01-02", "2022-12-13", time.UTC)
				Expect(err).NotTo(HaveOccurred())

				Expect(dependency).To(Equal(cargo.ConfigMetadataDependency{
					Checksum:        "sha512:365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6",
					CPE:             "cpe:2.3:a:microsoft:.net_core:3.1.423:*:*:*:*:*:*:*",
					PURL:            fmt.Sprintf("pkg:generic/dotnet-core-sdk@3.1.423?checksum=365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6&download_url=%s", server.URL),
					DeprecationDate: &depDate,
					ID:              "dotnet-core-sdk",
					Licenses:        []interface{}{"MIT", "MIT-0"},
					Name:            ".NET Core SDK",
					SHA256:          "",
					Source:          server.URL,
					SourceChecksum:  "sha512:365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6",
					SourceSHA256:    "",
					Stacks: []string{
						"io.buildpacks.stacks.bionic",
						"io.paketo.stacks.tiny",
					},
					StripComponents: 0,
					URI:             server.URL,
					Version:         "3.1.423",
				}))
			})
		})

		context("failure cases", func() {
			context("when there is not a linux-x64 release file", func() {
				it("returns an error", func() {
					_, err := components.ConvertReleaseToDependency(components.Release{})
					Expect(err).To(MatchError("could not find release file for linux/x64"))
				})
			})

			context("when license generation fails", func() {
				it("returns an error", func() {
					_, err := components.ConvertReleaseToDependency(components.Release{
						SemVer:  semver.MustParse("3.1.423"),
						EOLDate: "2022-12-13",
						Version: "3.1.423",
						Files: []components.ReleaseFile{
							{
								Name: "dotnet-sdk-linux-x64.tar.gz",
								Rid:  "linux-x64",
								URL:  fmt.Sprintf("%s/no-license", server.URL),
								Hash: "365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6",
							},
						},
					},
					)
					Expect(err).To(MatchError(ContainSubstring("no license file was found")))
				})
			})

			context("when the checksum does not match", func() {
				it("returns an error", func() {
					_, err := components.ConvertReleaseToDependency(components.Release{
						SemVer:  semver.MustParse("3.1.423"),
						EOLDate: "2022-12-13",
						Version: "3.1.423",
						Files: []components.ReleaseFile{
							{
								Name: "dotnet-sdk-linux-x64.tar.gz",
								Rid:  "linux-x64",
								URL:  server.URL,
								Hash: "invlaid hash",
							},
						},
					},
					)
					Expect(err).To(MatchError("the given checksum of the artifact does not match with downloaded artifact"))
				})
			})

			context("when the eol date is unparseable", func() {
				it("returns an error", func() {
					_, err := components.ConvertReleaseToDependency(components.Release{
						SemVer:  semver.MustParse("3.1.423"),
						EOLDate: "not a date",
						Version: "3.1.423",
						Files: []components.ReleaseFile{
							{
								Name: "dotnet-sdk-linux-x64.tar.gz",
								Rid:  "linux-x64",
								URL:  server.URL,
								Hash: "365237c83e7b0b836d933618bb8be9cee018e905b2c01156ef0ae1162cffbdc003ae4082ea9bb85d39f667e875882804c00d90a4280be4486ec81edb2fb64ad6",
							},
						},
					},
					)
					Expect(err).To(MatchError(ContainSubstring("cannot parse")))
				})
			})
		})
	})
}
