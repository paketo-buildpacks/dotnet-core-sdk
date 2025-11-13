package main

import (
	"github.com/paketo-buildpacks/dotnet-core-sdk/dependency/retrieval/components"
	"github.com/paketo-buildpacks/libdependency/retrieve"
)

func main() {
	fetcher := components.NewFetcher()
	retrieve.NewMetadataWithPlatforms("icu", fetcher.GetVersions, components.GenerateMetadata)
}
