package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
)

type BuildPlanRefinery struct {
	BillOfMaterialCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Dependency postal.Dependency
		}
		Returns struct {
			BuildpackPlanEntry packit.BuildpackPlanEntry
		}
		Stub func(postal.Dependency) packit.BuildpackPlanEntry
	}
}

func (f *BuildPlanRefinery) BillOfMaterial(param1 postal.Dependency) packit.BuildpackPlanEntry {
	f.BillOfMaterialCall.Lock()
	defer f.BillOfMaterialCall.Unlock()
	f.BillOfMaterialCall.CallCount++
	f.BillOfMaterialCall.Receives.Dependency = param1
	if f.BillOfMaterialCall.Stub != nil {
		return f.BillOfMaterialCall.Stub(param1)
	}
	return f.BillOfMaterialCall.Returns.BuildpackPlanEntry
}
