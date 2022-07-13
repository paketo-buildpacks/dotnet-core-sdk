package main

import (
	"os"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type Generator struct{}

func (f Generator) GenerateFromDependency(dependency postal.Dependency, path string) (sbom.SBOM, error) {
	return sbom.GenerateFromDependency(dependency, path)
}

func main() {
	sdkVersionParser := dotnetcoresdk.NewSdkVersionParser()
	logEmitter := scribe.NewEmitter(os.Stdout)
	entryResolver := draft.NewPlanner()
	dependencyManager := postal.NewService(cargo.NewTransport())

	packit.Run(
		dotnetcoresdk.Detect(sdkVersionParser),
		dotnetcoresdk.Build(
			entryResolver,
			dependencyManager,
			Generator{},
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
