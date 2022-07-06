package dotnetcoresdk

import "regexp"

var Priorities = []interface{}{
	"BP_DOTNET_FRAMEWORK_VERSION",
	"buildpack.yml",
	"global.json",
	regexp.MustCompile(`.*\.(cs)|(fs)|(vb)proj`),
	"runtimeconfig.json",
	"",
}

const (
	DotnetDependency = "dotnet-sdk"
)
