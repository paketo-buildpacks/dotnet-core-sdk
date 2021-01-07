package fakes

import "sync"

type DotnetSymlinker struct {
	LinkCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
			LayerPath  string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
}

func (f *DotnetSymlinker) Link(param1 string, param2 string) error {
	f.LinkCall.Lock()
	defer f.LinkCall.Unlock()
	f.LinkCall.CallCount++
	f.LinkCall.Receives.WorkingDir = param1
	f.LinkCall.Receives.LayerPath = param2
	if f.LinkCall.Stub != nil {
		return f.LinkCall.Stub(param1, param2)
	}
	return f.LinkCall.Returns.Error
}
