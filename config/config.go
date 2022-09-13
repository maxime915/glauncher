package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kirsle/configdir"
)

const LogToStderr = "--use-stderr"

type Config struct {
	// path to executables
	FzfPath string `json:"fzf-path"`

	// path to use for a log file
	LogFile string `json:"log-file"`

	/// Remote configuration

	// Name of the remote to use
	Selected string `json:"selected-remote"`
	// configs for all defined remote
	Remotes map[string]map[string]any `json:"remotes-configs"`

	/// Provider configuration

	// Providers in the blacklist will not be used
	Blacklist []string `json:"blacklist"`
	// configs for all defined provider
	Providers map[string]map[string]any `json:"providers-config"`

	ConfigFile string
}

func defaultConfig() *Config {
	// validation is used to set default fields
	return &Config{}
}

func readConfigAt(configFile string) (*Config, error) {
	err := os.MkdirAll(filepath.Dir(configFile), 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create config directory: %w", err)
	}

	if _, err = os.Stat(configFile); os.IsNotExist(err) {
		// create a new config file
		config := defaultConfig()
		config.ConfigFile = configFile

		err = config.Save()
		if err != nil {
			return nil, fmt.Errorf("unable to create config file: %w", err)
		}
		return config, nil
	} else {
		fh, err := os.Open(configFile)
		if err != nil {
			return nil, fmt.Errorf("unable to open config file: %w", err)
		}
		defer fh.Close()

		var config *Config
		decoder := json.NewDecoder(fh)
		err = decoder.Decode(&config)
		config.ConfigFile = configFile
		return config, err
	}
}

func (c *Config) Save() error {
	// temporary file to avoid corruption
	tempFile, err := os.CreateTemp("", "*.json")
	if err != nil {
		return err
	}

	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	// write to the temporary file
	encoder := json.NewEncoder(tempFile)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(c)
	if err != nil {
		return err
	}

	// move the temporary file to the real file (atomic)
	return os.Rename(tempFile.Name(), c.ConfigFile)
}

func LoadConfigAt(configFile string) (*Config, error) {
	config, err := readConfigAt(configFile)
	if err != nil {
		return nil, err
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	err = config.Save()
	return config, err
}

func (config *Config) validate() (err error) {
	if len(config.FzfPath) == 0 {
		config.FzfPath = "fzf"
	}

	// initialize map's

	if config.Remotes == nil {
		config.Remotes = make(map[string]map[string]any)
	}

	if config.Providers == nil {
		config.Providers = make(map[string]map[string]any)
	}

	return nil
}

func DefaultConfigPath() (string, error) {
	configPath := configdir.LocalConfig("glauncher")
	err := configdir.MakePath(configPath)
	if err != nil {
		return "", fmt.Errorf("unable to create config directory: %w", err)
	}

	configFile := filepath.Join(configPath, "config.json")

	return configFile, nil
}

func LoadConfig() (*Config, error) {
	if configFile, err := DefaultConfigPath(); err != nil {
		return nil, err
	} else {
		return LoadConfigAt(configFile)
	}
}
