package dotnetcoresdk

import "regexp"

var Priorities = []interface{}{
	// Roll-forward priority order from global.json
	"global.json exact",
	"global.json patch",
	"global.json feature",
	"global.json minor",
	"global.json major",

	// Other version sources
	"BP_DOTNET_FRAMEWORK_VERSION",
	"buildpack.yml",
	regexp.MustCompile(`.*\.(cs)|(fs)|(vb)proj`),
	"runtimeconfig.json",
	"",
}

const (
	DotnetDependency = "dotnet-sdk"
)
