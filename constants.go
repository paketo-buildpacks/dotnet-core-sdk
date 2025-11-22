package dotnetcoresdk

import "regexp"

var Priorities = []interface{}{
	"global.json",
	"BP_DOTNET_FRAMEWORK_VERSION",
	"buildpack.yml",
	regexp.MustCompile(`.*\.(cs)|(fs)|(vb)proj`),
	"runtimeconfig.json",
	"",
}

const (
	DotnetDependency = "dotnet-sdk"
)
