package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
)

type DependencyResolver struct {
	ResolveCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			CnbDir string
			Entry  packit.BuildpackPlanEntry
			Stack  string
		}
		Returns struct {
			Dependency postal.Dependency
			Error      error
		}
		Stub func(string, packit.BuildpackPlanEntry, string) (postal.Dependency, error)
	}
}

func (f *DependencyResolver) Resolve(param1 string, param2 packit.BuildpackPlanEntry, param3 string) (postal.Dependency, error) {
	f.ResolveCall.Lock()
	defer f.ResolveCall.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.CnbDir = param1
	f.ResolveCall.Receives.Entry = param2
	f.ResolveCall.Receives.Stack = param3
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1, param2, param3)
	}
	return f.ResolveCall.Returns.Dependency, f.ResolveCall.Returns.Error
}
