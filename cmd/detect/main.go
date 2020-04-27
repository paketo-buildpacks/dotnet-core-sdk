package main

import (
	"fmt"
	"os"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/paketo-buildpacks/dotnet-core-sdk/sdk"
)

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create a default detection context: %s", err)
		os.Exit(100)
	}

	code, err := runDetect(context)
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runDetect(context detect.Detect) (int, error) {
	plan := buildplan.Plan{
		Provides: []buildplan.Provided{{Name: sdk.DotnetSDK}},
	}

	runtimeConfig, err := sdk.NewRuntimeConfig(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	hasASPNetDependency := runtimeConfig.HasASPNetDependency()
	hasRuntimeDependency := runtimeConfig.HasRuntimeDependency()
	hasFDDependency := hasASPNetDependency || hasRuntimeDependency

	hasFDE, err := runtimeConfig.HasExecutable()
	if err != nil {
		return context.Fail(), err
	} else if !hasFDDependency || hasFDE {
		return context.Pass(plan)
	}

	plan.Requires = []buildplan.Required{{
		Name:     sdk.DotnetSDK,
		Version:  runtimeConfig.Version,
		Metadata: buildplan.Metadata{"build": true, "launch": true},
	}, {
		Name:     "dotnet-runtime",
		Version:  runtimeConfig.Version,
		Metadata: buildplan.Metadata{"build": true, "launch": true},
	}}

	if hasASPNetDependency {
		plan.Requires = append(plan.Requires, buildplan.Required{
			Name:     "dotnet-aspnetcore",
			Version:  runtimeConfig.Version,
			Metadata: buildplan.Metadata{"build": true, "launch": true},
		})
	}

	return context.Pass(plan)
}
