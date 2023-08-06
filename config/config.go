package config

/*
NOTE: to future readers
- The file lock is *not* reentrant and this can cause deadlocks.
- Exported methods must lock the file before reading or writings.
- private methods must *not* lock the file, if access to the file is necessary, the lock must be passed as an argument
- As such, exported method must *not* call other exported methods if perform any locking.
*/

// While the config file can be read, written to, and truncated while a process of glauncher (server, client, or cli) is running, it should not be unlinked as this may cause issues. Why ? dunno...

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/gofrs/flock"
	"github.com/kirsle/configdir"
)

const LogToStderr = "--use-stderr"

var (
	errNotLocked = fmt.Errorf("config should be locked")
)

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
	Blacklist []string `json:"providers-blacklist"`
	// configs for all defined provider
	Providers map[string]map[string]any `json:"providers-config"`

	// path to the config file: not saved
	ConfigFile string `json:"-"`
}

func defaultConfig() *Config {
	// validation is used to set default fields
	return &Config{}
}

// lock returns a *locked* file lock on the configPath,
// creating it and directories with correct permissions if necessary
func lock(configPath string) (*flock.Flock, error) {
	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create config directory: %w", err)
	}

	lock := flock.New(configPath)
	return lock, lock.Lock()
}

func readConfigAt(lock *flock.Flock) (*Config, error) {
	if !lock.Locked() {
		return nil, errNotLocked
	}
	configFile := lock.Path()

	// the file should always exist: we've locked it
	fStat, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("locked file doesn't exist: %w", err)
	} else if err != nil {
		return nil, err
	}

	// check for empty file
	if fStat.Size() == 0 {
		// create a new config file
		config := defaultConfig()
		config.ConfigFile = configFile

		err = config.rawSave(lock)
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

		var config Config
		decoder := json.NewDecoder(fh)
		err = decoder.Decode(&config)
		config.ConfigFile = configFile
		return &config, err
	}
}

func (c *Config) rawSave(lock *flock.Flock) error {
	if !lock.Locked() {
		return errNotLocked
	}

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
	lock, err := lock(configFile)
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	config, err := rawRead(lock)
	if err != nil {
		return nil, err
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	err = config.saveIfChanged(lock)
	return config, err
}

func rawRead(lock *flock.Flock) (*Config, error) {
	if !lock.Locked() {
		return nil, errNotLocked
	}

	config, err := readConfigAt(lock)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Save() error {
	lock, err := lock(c.ConfigFile)
	if err != nil {
		return err
	}
	defer lock.Unlock()

	return c.saveIfChanged(lock)
}

func (c *Config) saveIfChanged(lock *flock.Flock) error {
	// make sure the copy is live
	latest, err := rawRead(lock)
	if err == errNotLocked {
		panic("expected lock to be locked")
	} else if err != nil {
		return err
	}

	// if they are equal, no need for an update
	if reflect.DeepEqual(*c, *latest) {
		return nil
	}

	// some differences have been found, update the file
	return c.rawSave(lock)
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
