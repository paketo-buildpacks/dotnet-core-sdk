package main

import (
	"fmt"
	"github.com/cloudfoundry/dotnet-core-conf-cnb/utils"
	"github.com/cloudfoundry/dotnet-core-sdk-cnb/sdk"
	"github.com/pkg/errors"
	"os"
	"strings"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
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

	runtimeConfig, err := utils.NewRuntimeConfig(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	hasASPNetDependency := runtimeConfig.HasASPNetDependency()
	hasRuntimeDependency := runtimeConfig.HasRuntimeDependency()
	hasFDDependency := hasASPNetDependency || hasRuntimeDependency

	hasFDE, err := runtimeConfig.HasFDE()
	if err != nil{
		return context.Fail(), err
	} else if !hasFDDependency || hasFDE {
		return context.Pass(plan)
	}

	runtimeVersion := runtimeConfig.Version

	version, majorMinor, err := getLatestCompatibleSDK(runtimeVersion, context)
	if err != nil {
		return context.Fail(), err
	}

	plan.Requires = []buildplan.Required{{
		Name:     sdk.DotnetSDK,
		Version:  version,
		Metadata: buildplan.Metadata{"launch": true},
	},{
		Name:     "dotnet-runtime",
		Version:  majorMinor,
		Metadata: buildplan.Metadata{"build": true, "launch": true},
	}}

	if hasASPNetDependency{
		plan.Requires = append(plan.Requires, buildplan.Required{
			Name:     "dotnet-aspnet",
			Version:  majorMinor,
			Metadata: buildplan.Metadata{"build": true, "launch": true},
		})
	}

	return context.Pass(plan)
}



func getLatestCompatibleSDK(frameworkVersion string, context detect.Detect) (string, string, error){
	splitVersion := strings.Split(frameworkVersion, ".")
	compatibleVersionConstraint := fmt.Sprintf("%s.%s.*", splitVersion[0], splitVersion[1])

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return "", "", err
	}

	compatibleVersion, err := deps.Best(sdk.DotnetSDK, compatibleVersionConstraint, context.Stack)

	if err != nil {
		return "", "", errors.Wrap(err, "no compatible version of the sdk found")
	}

	return compatibleVersion.Version.Original(), compatibleVersionConstraint, nil
}

