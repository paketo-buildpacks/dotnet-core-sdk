package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/gravityblast/go-jsmin"
	"github.com/pkg/errors"
)

type RuntimeConfig struct {
	isPresent  bool
	config     configJSON
	appRoot    string
	BinaryName string
	Version    string
}

type configJSON struct {
	RuntimeOptions struct {
		Framework struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"framework"`
		ApplyPatches bool `json:"applyPatches"`
	} `json:"runtimeOptions"`
}

func NewRuntimeConfig(appRoot string) (*RuntimeConfig, error) {
	runtimeConfigPath, err := getRuntimeConfigPath(appRoot)
	if err != nil {
		return &RuntimeConfig{}, err
	} else if runtimeConfigPath == "" {
		return &RuntimeConfig{}, nil
	}

	config, err := parseRuntimeConfig(runtimeConfigPath)
	if err != nil {
		return &RuntimeConfig{}, err
	}

	return &RuntimeConfig{
		isPresent:  true,
		config:     config,
		appRoot:    appRoot,
		BinaryName: getBinaryName(runtimeConfigPath),
		Version:    config.RuntimeOptions.Framework.Version,
	}, nil
}

func (r *RuntimeConfig) IsPresent() bool {
	return r.isPresent
}

func (r *RuntimeConfig) HasRuntimeDependency() bool {
	return r.config.RuntimeOptions.Framework.Name == "Microsoft.NETCore.App"
}

func (r *RuntimeConfig) HasASPNetDependency() bool {
	return r.config.RuntimeOptions.Framework.Name == "Microsoft.AspNetCore.App" ||
		r.config.RuntimeOptions.Framework.Name == "Microsoft.AspNetCore.All"
}

func (r *RuntimeConfig) HasApplyPatches() bool {
	return r.config.RuntimeOptions.ApplyPatches
}

func getBinaryName(runtimeConfigPath string) string {
	runtimeConfigFile := filepath.Base(runtimeConfigPath)
	executableFile := strings.ReplaceAll(runtimeConfigFile, ".runtimeconfig.json", "")

	return executableFile
}

func (r *RuntimeConfig) HasExecutable() (bool, error) {
	exists, err := helper.FileExists(filepath.Join(r.appRoot, r.BinaryName))
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	info, err := os.Stat(filepath.Join(r.appRoot, r.BinaryName))
	if err != nil {
		return false, err
	}

	if info.Mode()&0111 != 0 {
		return true, nil
	}

	return false, nil
}

func getRuntimeConfigPath(appRoot string) (string, error) {
	if configFiles, err := filepath.Glob(filepath.Join(appRoot, "*.runtimeconfig.json")); err != nil {
		return "", err
	} else if len(configFiles) == 1 {
		return configFiles[0], nil
	} else if len(configFiles) > 1 {
		return "", fmt.Errorf("multiple *.runtimeconfig.json files present")
	}
	return "", nil
}

func parseRuntimeConfig(runtimeConfigPath string) (configJSON, error) {
	obj := configJSON{}

	buf, err := sanitizeJsonConfig(runtimeConfigPath)
	if err != nil {
		return obj, err
	}

	if err := json.Unmarshal(buf, &obj); err != nil {
		return obj, errors.Wrap(err, "unable to parse runtime config")
	}

	return obj, nil
}

func sanitizeJsonConfig(runtimeConfigPath string) ([]byte, error) {
	input, err := os.Open(runtimeConfigPath)
	if err != nil {
		return nil, err
	}
	defer input.Close()

	output := &bytes.Buffer{}

	if err := jsmin.Min(input, output); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}
