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
	suite("GlobalFileParser", testGlobalFileParser)
	suite("RollforwardResolver", testRollforwardResolver)
	suite.Run(t)
}
