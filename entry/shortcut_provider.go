package entry

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	config "github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/frontend"
	"github.com/maxime915/glauncher/utils"
)

const ShortCutProviderKey = "shortcut-provider"

var (
	ErrInvalidScheme = errors.New("forbidden scheme in URL")
	// Empty scheme relates to files
	allowedScheme = []string{"", "http", "https", "file"}
)

func validateURL(path string) error {
	url, err := url.ParseRequestURI(string(path))
	if err != nil {
		return err
	}

	isAllowed := false
	for _, scheme := range allowedScheme {
		if scheme == url.Scheme {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return ErrInvalidScheme
	}

	return nil
}

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

func defaultShortcutList() shortcutList {
	return map[string]ShortCut{}
}

func AddShortcutsToConfig(conf *config.Config, shortcuts map[string]ShortCut, override bool) error {
	// get current commands
	currentShortcuts, err := utils.ValFromJSON[shortcutList](conf.Providers[ShortCutProviderKey])
	if err != nil {
		return nil
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

		if strings.HasPrefix(string(v), "~/") {
			path, err := utils.ResolvePath(string(v))
			if err != nil {
				return err
			}
			v = ShortCut(path)
		}

		err := validateURL(string(v))
		if err != nil {
			return err
		}

		currentShortcuts[k] = v
	}

	// save shortcuts
	shortcutsSerialized, err := utils.ValToJSON(currentShortcuts)
	if err != nil {
		return err
	}

	conf.Providers[ShortCutProviderKey] = shortcutsSerialized
	return conf.Save()
}

func NewShortcutProvider(conf *config.Config, options map[string]string) (EntryProvider, error) {
	var err error

	// parse shortcuts
	var shortcuts shortcutList
	shortcutsStr := conf.Providers[ShortCutProviderKey]
	if len(shortcutsStr) == 0 {
		// get the defaults, and store them
		shortcuts = defaultShortcutList()
		err = AddShortcutsToConfig(conf, shortcuts, false)
		if err != nil {
			return nil, err
		}
	} else {
		err = utils.FromJSON(shortcutsStr, &shortcuts)
		if err != nil {
			return nil, err
		}
	}

	// add config (if not already present)
	if _, ok := shortcuts["config"]; !ok {
		shortcuts["config"] = ShortCut(conf.ConfigFile)
	}

	for _, path := range shortcuts {
		err = validateURL(string(path))
		if err != nil {
			return nil, err
		}
	}

	return ShortCutProvider{
		Content: shortcuts,
		Prefix:  "& ",
	}, nil
}
