package fakes

import "sync"

type DependencyMapper struct {
	FindCorrespondingVersionCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path       string
			VersionKey string
		}
		Returns struct {
			String string
			Error  error
		}
		Stub func(string, string) (string, error)
	}
}

func (f *DependencyMapper) FindCorrespondingVersion(param1 string, param2 string) (string, error) {
	f.FindCorrespondingVersionCall.Lock()
	defer f.FindCorrespondingVersionCall.Unlock()
	f.FindCorrespondingVersionCall.CallCount++
	f.FindCorrespondingVersionCall.Receives.Path = param1
	f.FindCorrespondingVersionCall.Receives.VersionKey = param2
	if f.FindCorrespondingVersionCall.Stub != nil {
		return f.FindCorrespondingVersionCall.Stub(param1, param2)
	}
	return f.FindCorrespondingVersionCall.Returns.String, f.FindCorrespondingVersionCall.Returns.Error
}
