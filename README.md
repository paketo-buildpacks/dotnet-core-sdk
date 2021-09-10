# Dotnet Core SDK Cloud Native Buildpack



The Dotnet Core SDK CNB provides a version of the Dotnet Core SDK and a version of the
Dotnet Core Driver or the `dotnet` binary. It also sets the Dotnet Core SDK on the`$DOTNET_ROOT`
so that it is available to subsequent buildpacks during their build phase and sets the Dotnet Core
Driver on the `$PATH` so that is available to subsequent buildpacks and in the final running container.

## Integration

The Dotnet Core SDK CNB provides the dotnet-sdk as a dependency. Downstream buildpacks, like
[Dotnet Core Build](https://github.com/paketo-buildpacks/dotnet-core-build) can
require the dotnet-sdk dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the Dotnet Core SDK dependency is "dotnet-sdk". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "dotnet-sdk"

  # The version of the Dotnet Core SDK dependency is not required. In the case it
  # is not specified, the buildpack will provide the default version, which can
  # be seen in the buildpack.toml file.
  # If you wish to request a specific version, the buildpack supports
  # specifying a semver constraint in the form of "3.*", "3.1.*", or even
  # "3.1.100".
  version = "3.1.100"

  # The Dotnet Core SDK buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the Dotnet Core SDK
    # depdendency is available in the $DOTNET_ROOT for subsequent buildpacks during
    # their build phase and ensures that the Dotnet Core Driver dependency is available
    # in the $PATH for subsequent buildpacks. If you are writing a buildpack that needs
    # to use the dotnet sdk & driver during its build process, this flag should be set
    # to true.
    build = true
```

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh
```

This builds the buildpack's Go source using `GOOS=linux` by default. You can
supply another value as the first argument to `package.sh`.

## (Deprecated) `buildpack.yml` Configurations

```yaml
dotnet-sdk:
  # this allows you to specify a version constaint for the dotnet-sdk dependency
  # any valid semver constaints (e.g. 2.* and 2.1.*) are also acceptable
  version: "2.1.804"
```
This configuration option will be deprecated with the next major version
release of the buildpack. Because the versions of the .NET Core runtime and
.NET Core SDK are so tightly coupled, most users should instead use the
`$BP_DOTNET_FRAMEWORK_VERSION` environment variable to specify which version of
the .NET Core runtime that the [Paketo .NET Core Runtime
Buildpack](https://github.com/paketo-buildpacks/dotnet-core-runtime) should
install. This buildpack will automatically select an SDK version to install
that is compatible with the selected .NET Core runtime version.
