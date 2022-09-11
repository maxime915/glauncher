package entry

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"

	config "github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/frontend"
)

const ShortCutProviderKey = "shortcuts"

var (
	ErrInvalidScheme = errors.New("forbidden scheme in URL")
	// Empty scheme relates to files
	allowedScheme = []string{"", "http", "https", "file"}
)

// either an URI or an absolute path
type ShortCut string

func init() {
	RegisterEntryType[ShortCut]()
	registerProvider(ShortCutProviderKey, NewShortcutProvider)
}

func (s ShortCut) LaunchInFrontend(_ frontend.Frontend, _ map[string]string) error {
	return ErrRemoteRequired
}

func (s ShortCut) RemoteLaunch(options map[string]string) error {
	// open uri pointed to by to the shortcut
	cmd := exec.Command("xdg-open", string(s))
	return cmd.Run()
}

// provide URIs and path as shortcuts
type ShortCutProvider = MapProvider[ShortCut]

// struct to store the commands in the config files
type shortcutList = map[string]ShortCut

func defaultShortcutList() (shortcutList, error) {
	configFile, err := config.DefaultConfigPath()
	if err != nil {
		return shortcutList{}, err
	}

	return map[string]ShortCut{"config": ShortCut(configFile)}, nil
}

func AddShortcutsToConfig(conf *config.Config, shortcuts map[string]ShortCut, override bool) error {
	// get current commands
	var currentShortcuts shortcutList
	shortcutStr := conf.Providers[ShortCutProviderKey]
	if len(shortcutStr) != 0 {
		err := json.Unmarshal([]byte(shortcutStr), &currentShortcuts)
		if err != nil {
			return nil
		}
	}
	if currentShortcuts == nil {
		currentShortcuts = make(shortcutList, len(shortcuts))
	}

	// check for overriding
	if !override {
		var duplicates []string
		// check for duplicates
		for k := range shortcuts {
			if _, ok := currentShortcuts[k]; ok {
				duplicates = append(duplicates, k)
			}
		}

		if len(duplicates) > 0 {
			return fmt.Errorf("duplicate shortcuts: %s", duplicates)
		}
	}

	// merge shortcuts
	for k, v := range shortcuts {
		currentShortcuts[k] = v
	}

	// save shortcuts
	shortcutsSerialized, err := json.Marshal(currentShortcuts)
	if err != nil {
		return err
	}

	conf.Providers[ShortCutProviderKey] = string(shortcutsSerialized)
	return conf.Save()
}

func NewShortcutProvider(conf *config.Config, options map[string]string) (EntryProvider, error) {
	var err error

	// parse shortcuts
	var shortcuts shortcutList
	shortcutsStr := conf.Providers[ShortCutProviderKey]
	if len(shortcutsStr) == 0 {
		// get the defaults, and store them
		shortcuts, err = defaultShortcutList()
		if err != nil {
			return nil, err
		}

		shortcutsBytes, err := json.Marshal(shortcuts)
		if err != nil {
			return nil, err
		}

		conf.Providers[ShortCutProviderKey] = string(shortcutsBytes)
		if err = conf.Save(); err != nil {
			return nil, err
		}
	} else {
		err = json.Unmarshal([]byte(shortcutsStr), &shortcuts)
		if err != nil {
			return nil, err
		}
	}

	for _, path := range shortcuts {
		// accepts http URI and absolute files
		url, err := url.ParseRequestURI(string(path))
		if err != nil {
			return nil, err
		}
		isAllowed := false
		for _, scheme := range allowedScheme {
			if scheme == url.Scheme {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return nil, ErrInvalidScheme
		}
	}

	return ShortCutProvider{
		Content: shortcuts,
		Prefix:  "🔗 ",
	}, nil
}
