package main

import (
	"os"

	dotnetcoresdk "github.com/paketo-buildpacks/dotnet-core-sdk"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
)

func main() {
	sdkVersionParser := dotnetcoresdk.NewSdkVersionParser()
	logEmitter := dotnetcoresdk.NewLogEmitter(os.Stdout)
	entryResolver := dotnetcoresdk.NewPlanEntryResolver(logEmitter)
	dependencyResolver := dotnetcoresdk.NewSDKVersionResolver(logEmitter)
	dependencyManager := postal.NewService(cargo.NewTransport())
	planRefinery := dotnetcoresdk.NewPlanRefinery()
	symlinker := dotnetcoresdk.NewSymlinker()

	packit.Run(
		dotnetcoresdk.Detect(sdkVersionParser),
		dotnetcoresdk.Build(
			entryResolver,
			dependencyResolver,
			planRefinery,
			dependencyManager,
			symlinker,
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
