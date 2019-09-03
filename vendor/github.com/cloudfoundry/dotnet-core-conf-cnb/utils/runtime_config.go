package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/gravityblast/go-jsmin"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"
)

type RuntimeConfig struct {
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
	} `json:"runtimeOptions"`
}

func NewRuntimeConfig(appRoot string) (*RuntimeConfig, error) {
	runtimeConfigPath, err := getRuntimeConfigPath(appRoot)
	if err != nil {
		return &RuntimeConfig{}, err
	}

	if runtimeConfigPath == "" {
		return &RuntimeConfig{}, nil
	}

	config, err := createRuntimeConfig(runtimeConfigPath)
	if err != nil {
		return &RuntimeConfig{}, err
	}

	return &RuntimeConfig{
		config:     config,
		appRoot:    appRoot,
		BinaryName: getBinaryName(runtimeConfigPath),
		Version:    config.RuntimeOptions.Framework.Version,
	}, nil
}

func (r *RuntimeConfig) HasRuntimeDependency() bool {
	return r.config.RuntimeOptions.Framework.Name == "Microsoft.NETCore.App"
}

func (r *RuntimeConfig) HasASPNetDependency() bool {

	return r.config.RuntimeOptions.Framework.Name == "Microsoft.AspNetCore.App" || r.config.RuntimeOptions.Framework.Name == "Microsoft.AspNetCore.All"
}

func getBinaryName(runtimeConfigPath string) string {

	runtimeConfigFile := filepath.Base(runtimeConfigPath)
	executableFile := strings.ReplaceAll(runtimeConfigFile, ".runtimeconfig.json", "")

	return executableFile
}

func (r *RuntimeConfig) HasFDE() (bool, error) {

	exists, err := helper.FileExists(filepath.Join(r.appRoot, r.BinaryName))
	if err != nil {
		return false, err
	}

	executable := false

	if exists {
		executable, err = isExecutable(filepath.Join(r.appRoot, r.BinaryName))
		if err != nil {
			return false, err
		}
	}

	return executable, nil
}

func isExecutable(fileName string) (bool, error) {
	info, err := os.Stat(fileName)
	if err != nil {
		return false, err
	}
	if info.Mode()&0111 != 0 {
		return true, nil
	}
	return false, nil
}

func createRuntimeConfig(path string) (configJSON, error) {
	var err error
	runtimeJSON := configJSON{}

	if path != "" {
		runtimeJSON, err = parseRuntimeConfig(path)
		if err != nil {
			return configJSON{}, err
		}
	}

	return runtimeJSON, nil
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