package dotnetcoresdk_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDotnetCoreSDK(t *testing.T) {
	suite := spec.New("dotnet-core-sdk", spec.Report(report.Terminal{}), spec.Sequential())
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite("LogEmitter", testLogEmitter)
	suite("PlanRefinery", testPlanRefinery)
	suite("SDKVersionMapper", testSDKVersionMapper)
	suite("SdkVersionParser", testSdkVersionParser)
	suite("Symlinker", testSymlinker)
	suite.Run(t)
}
