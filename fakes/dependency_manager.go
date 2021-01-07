package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit/postal"
)

type DependencyManager struct {
	InstallCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Dependency postal.Dependency
			CnbPath    string
			LayerPath  string
		}
		Returns struct {
			Error error
		}
		Stub func(postal.Dependency, string, string) error
	}
}

func (f *DependencyManager) Install(param1 postal.Dependency, param2 string, param3 string) error {
	f.InstallCall.Lock()
	defer f.InstallCall.Unlock()
	f.InstallCall.CallCount++
	f.InstallCall.Receives.Dependency = param1
	f.InstallCall.Receives.CnbPath = param2
	f.InstallCall.Receives.LayerPath = param3
	if f.InstallCall.Stub != nil {
		return f.InstallCall.Stub(param1, param2, param3)
	}
	return f.InstallCall.Returns.Error
}
