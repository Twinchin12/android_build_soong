// Copyright 2015 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// The configuration file name
const ConfigFileName = "soong.config"
const ProductVariablesFileName = "soong.variables"

// A FileConfigurableOptions contains options which can be configured by the
// config file. These will be included in the config struct.
type FileConfigurableOptions struct {
}

func (FileConfigurableOptions) DefaultConfig() jsonConfigurable {
	f := FileConfigurableOptions{}
	return f
}

type Config struct {
	*config
}

// A config object represents the entire build configuration for Blue.
type config struct {
	FileConfigurableOptions
	ProductVariables productVariables

	srcDir string // the path of the root source directory

	envLock sync.Mutex
	envDeps map[string]string
}

type jsonConfigurable interface {
	DefaultConfig() jsonConfigurable
}

func loadConfig(config *config) error {
	err := loadFromConfigFile(&config.FileConfigurableOptions, ConfigFileName)
	if err != nil {
		return err
	}

	return loadFromConfigFile(&config.ProductVariables, ProductVariablesFileName)
}

// loads configuration options from a JSON file in the cwd.
func loadFromConfigFile(configurable jsonConfigurable, filename string) error {
	// Try to open the file
	configFileReader, err := os.Open(filename)
	defer configFileReader.Close()
	if os.IsNotExist(err) {
		// Need to create a file, so that blueprint & ninja don't get in
		// a dependency tracking loop.
		// Make a file-configurable-options with defaults, write it out using
		// a json writer.
		err = saveToConfigFile(configurable.DefaultConfig(), filename)
		if err != nil {
			return err
		}
	} else {
		// Make a decoder for it
		jsonDecoder := json.NewDecoder(configFileReader)
		err = jsonDecoder.Decode(configurable)
		if err != nil {
			return fmt.Errorf("config file: %s did not parse correctly: "+err.Error(), filename)
		}
	}

	// No error
	return nil
}

func saveToConfigFile(config jsonConfigurable, filename string) error {
	data, err := json.MarshalIndent(&config, "", "    ")
	if err != nil {
		return fmt.Errorf("cannot marshal config data: %s", err.Error())
	}

	configFileWriter, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("cannot create empty config file %s: %s\n", filename, err.Error())
	}
	defer configFileWriter.Close()

	_, err = configFileWriter.Write(data)
	if err != nil {
		return fmt.Errorf("default config file: %s could not be written: %s", filename, err.Error())
	}

	_, err = configFileWriter.WriteString("\n")
	if err != nil {
		return fmt.Errorf("default config file: %s could not be written: %s", filename, err.Error())
	}

	return nil
}

// New creates a new Config object.  The srcDir argument specifies the path to
// the root source directory. It also loads the config file, if found.
func NewConfig(srcDir string) (Config, error) {
	// Make a config with default options
	config := Config{
		config: &config{
			srcDir:  srcDir,
			envDeps: make(map[string]string),
		},
	}

	// Load any configurable options from the configuration file
	err := loadConfig(config.config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func (c *config) SrcDir() string {
	return c.srcDir
}

func (c *config) IntermediatesDir() string {
	return ".intermediates"
}

// PrebuiltOS returns the name of the host OS used in prebuilts directories
func (c *config) PrebuiltOS() string {
	switch runtime.GOOS {
	case "linux":
		return "linux-x86"
	case "darwin":
		return "darwin-x86"
	default:
		panic("Unknown GOOS")
	}
}

// GoRoot returns the path to the root directory of the Go toolchain.
func (c *config) GoRoot() string {
	return fmt.Sprintf("%s/prebuilts/go/%s", c.srcDir, c.PrebuiltOS())
}

func (c *config) CpPreserveSymlinksFlags() string {
	switch runtime.GOOS {
	case "darwin":
		return "-R"
	case "linux":
		return "-d"
	default:
		return ""
	}
}

func (c *config) Getenv(key string) string {
	var val string
	var exists bool
	c.envLock.Lock()
	if val, exists = c.envDeps[key]; !exists {
		val = os.Getenv(key)
		c.envDeps[key] = val
	}
	c.envLock.Unlock()
	return val
}

func (c *config) EnvDeps() map[string]string {
	return c.envDeps
}

// DeviceName returns the name of the current device target
// TODO: take an AndroidModuleContext to select the device name for multi-device builds
func (c *config) DeviceName() string {
	return "unset"
}

// DeviceOut returns the path to out directory for device targets
func (c *config) DeviceOut() string {
	return filepath.Join("target/product", c.DeviceName())
}

// HostOut returns the path to out directory for host targets
func (c *config) HostOut() string {
	return filepath.Join("host", c.PrebuiltOS())
}

// HostBin returns the path to bin directory for host targets
func (c *config) HostBin() string {
	return filepath.Join(c.HostOut(), "bin")
}

// HostBinTool returns the path to a host tool in the bin directory for host targets
func (c *config) HostBinTool(tool string) (string, error) {
	return filepath.Join(c.HostBin(), tool), nil
}

// HostJavaDir returns the path to framework directory for host targets
func (c *config) HostJavaDir() string {
	return filepath.Join(c.HostOut(), "framework")
}

// HostJavaTool returns the path to a host tool in the frameworks directory for host targets
func (c *config) HostJavaTool(tool string) (string, error) {
	return filepath.Join(c.HostJavaDir(), tool), nil
}

func (c *config) ResourceOverlays() []string {
	return nil
}

func (c *config) PlatformVersion() string {
	return "M"
}

func (c *config) PlatformSdkVersion() string {
	return "22"
}

func (c *config) BuildNumber() string {
	return "000000"
}

func (c *config) ProductAaptConfig() []string {
	return []string{"normal", "large", "xlarge", "hdpi", "xhdpi", "xxhdpi"}
}

func (c *config) ProductAaptPreferredConfig() string {
	return "xhdpi"
}

func (c *config) ProductAaptCharacteristics() string {
	return "nosdcard"
}

func (c *config) DefaultAppCertificateDir() string {
	return filepath.Join(c.SrcDir(), "build/target/product/security")
}

func (c *config) DefaultAppCertificate() string {
	return filepath.Join(c.DefaultAppCertificateDir(), "testkey")
}
