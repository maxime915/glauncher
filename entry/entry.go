package entry

import (
	"errors"
	"io"

	"github.com/maxime915/glauncher/config"
	frontend "github.com/maxime915/glauncher/frontend"
)

type Entry interface {
	// LaunchInFrontend launches the entry in the provided frontend
	LaunchInFrontend(f frontend.Frontend, options map[string]string) error
	// RemoteLaunch should be used by the remote to launch the entry in a different process
	RemoteLaunch(options map[string]string) error
}

type EntryProvider interface {
	// returns a reader from which all keywords can be read
	GetEntryReader() (io.Reader, error)
	// returns a value for an entry
	Fetch(entry string) (Entry, bool)
}

type NewEntryProviderFun = func(*config.Config, map[string]string) (EntryProvider, error)

var (
	ErrNotFound         = errors.New("entry not found in this provider")
	ErrRemoteRequired   = errors.New("a remote is required for this entry")
	registeredProviders = make(map[string]NewEntryProviderFun)
)

func GetRegisteredProviderFun() map[string]NewEntryProviderFun {
	// return a copy to avoid modification
	copy := make(map[string]NewEntryProviderFun, len(registeredProviders))
	for k, v := range registeredProviders {
		copy[k] = v
	}
	return copy
}

func registerProvider(name string, providerFun NewEntryProviderFun) {
	registeredProviders[name] = providerFun
}

func GetProviders(
	conf *config.Config,
	options map[string]string,
	builders ...NewEntryProviderFun,
) ([]EntryProvider, error) {
	providers := make([]EntryProvider, len(builders))

	for i, builder := range builders {
		provider, err := builder(conf, options)
		if err != nil {
			return nil, err
		}

		providers[i] = provider
	}

	return providers, nil
}
