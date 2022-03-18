package main

import (
	"os"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/postal"
)

func main() {
	sdkVersionParser := dotnetcoresdk.NewSdkVersionParser()
	logEmitter := dotnetcoresdk.NewLogEmitter(os.Stdout)
	entryResolver := draft.NewPlanner()
	dependencyMapper := dotnetcoresdk.NewSDKVersionMapper(logEmitter)
	dependencyManager := postal.NewService(cargo.NewTransport())
	symlinker := dotnetcoresdk.NewSymlinker()

	packit.Run(
		dotnetcoresdk.Detect(sdkVersionParser),
		dotnetcoresdk.Build(
			entryResolver,
			dependencyMapper,
			dependencyManager,
			symlinker,
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
