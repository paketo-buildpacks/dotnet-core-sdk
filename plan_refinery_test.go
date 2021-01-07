package dotnetcoresdk_test

import (
	"testing"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPlanRefinery(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		refinery dotnetcoresdk.PlanRefinery
	)

	it.Before(func() {
		refinery = dotnetcoresdk.NewPlanRefinery()
	})

	context("BillOfMaterial", func() {
		it("returns a refined build plan entry", func() {
			entry := refinery.BillOfMaterial(postal.Dependency{
				ID:      "some-id",
				Name:    "some-name",
				Stacks:  []string{"some-stack"},
				URI:     "some-uri",
				SHA256:  "some-sha",
				Version: "some-version",
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "some-id",
				Metadata: map[string]interface{}{
					"licenses": []string{},
					"name":     "some-name",
					"sha256":   "some-sha",
					"stacks":   []string{"some-stack"},
					"uri":      "some-uri",
					"version":  "some-version",
				},
			}))
		})
	})
}
