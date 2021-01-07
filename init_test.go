package dotnetcoresdk_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDotnetCoreSDK(t *testing.T) {
	suite := spec.New("dotnet-core-sdk", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Detect", testDetect)
	suite("Build", testBuild)
	suite("SdkVersionParser", testSdkVersionParser)
	suite("PlanEntryResolver", testPlanEntryResolver)
	suite("LogEmitter", testLogEmitter)
	suite("PlanRefinery", testPlanRefinery)
	suite("SDKVersionResolver", testSDKVersionResolver)
	suite("Symlinker", testSymlinker)
	suite.Run(t)
}
