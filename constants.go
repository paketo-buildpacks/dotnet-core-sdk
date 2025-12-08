package dotnetcoresdk

import "regexp"

var Priorities = []interface{}{
	DotnetSdkVersion,
	DeprecatedFrameworkVersion,
	"buildpack.yml",
	"global.json",
	regexp.MustCompile(`.*\.(cs)|(fs)|(vb)proj`),
	"runtimeconfig.json",
	"",
}

const (
	DotnetDependency           = "dotnet-sdk"
	DotnetSdkVersion           = "BP_DOTNET_SDK_VERSION"
	DeprecatedFrameworkVersion = "BP_DOTNET_FRAMEWORK_VERSION"
)
