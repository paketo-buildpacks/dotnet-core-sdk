# .NET Core SDK Cloud Native Buildpack

The .NET Core SDK CNB provides a version of the .NET Core SDK and a version of the
.NET Core Driver or the `dotnet` binary. It also sets the .NET Core SDK on the`$DOTNET_ROOT`
so that it is available to subsequent buildpacks during their build phase and sets the .NET Core
Driver on the `$PATH` so that is available to subsequent buildpacks and in the final running container.

## Integration

The .NET Core SDK CNB provides the dotnet-sdk as a dependency. Downstream buildpacks, like
[.NET Core Build](https://github.com/paketo-buildpacks/dotnet-core-build) can
require the dotnet-sdk dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the .NET Core SDK dependency is "dotnet-sdk". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "dotnet-sdk"

  # The version of the .NET Core SDK dependency is not required. In the case it
  # is not specified, the buildpack will provide the default version, which can
  # be seen in the buildpack.toml file.
  # If you wish to request a specific version, the buildpack supports
  # specifying a semver constraint in the form of "6.*", "6.0.*", or even
  # "6.0.1".
  version = "6.0.1"

  # The .NET Core SDK buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the .NET Core SDK
    # depdendency is available in the $DOTNET_ROOT for subsequent buildpacks during
    # their build phase and ensures that the .NET Core Driver dependency is available
    # in the $PATH for subsequent buildpacks. If you are writing a buildpack that needs
    # to use the dotnet sdk & driver during its build process, this flag should be set
    # to true.
    build = true
```

## Configuration

### `BP_DOTNET_SDK_VERSION`
The `BP_DOTNET_SDK_VERSION` variable allows you to specify the version of
SDK that is installed. This overrides any configuration set in `global.json`.
The environment variable can be set at build-time either directly
(ex. `pack build my-app --env BP_ENVIRONMENT_VARIABLE=some-value`) or through a [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md)

```shell
BP_DOTNET_SDK_VERSION=8.0.*
```

### `BP_LOG_LEVEL`
The `BP_LOG_LEVEL` variable allows you to configure the level of log output
from the **buildpack itself**.  The environment variable can be set at build
time either directly (ex. `pack build my-app --env BP_LOG_LEVEL=DEBUG`) or
through a [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md)
If no value is set, the default value of `INFO` will be used.

The options for this setting are:
- `INFO`: (Default) log information about the progress of the build process
- `DEBUG`: log debugging information about the progress of the build process

```shell
BP_LOG_LEVEL="DEBUG"
```

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh
```

This builds the buildpack's Go source using `GOOS=linux` by default. You can
supply another value as the first argument to `package.sh`.
