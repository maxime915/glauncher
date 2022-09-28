package entry

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/frontend"
	"github.com/maxime915/glauncher/utils"
)

const (
	PathProviderKey     = "path-provider"
	OptionBaseDirectory = "base-directory"
	OptionHideFiles     = "hide-files"
	OptionHideFolders   = "hide-folders"
	OptionIgnoreVCS     = "no-ignore-vcs"
)

var (
	ErrKeyNotHandled = fmt.Errorf("key not handled")
)

// absolute path to open with xdg-open
type Path string

func init() {
	RegisterEntryType[Path]()
	registerProvider(PathProviderKey, NewPathProvider)
}

func (p Path) LaunchInFrontend(_ frontend.Frontend, options map[string]string) error {
	return ErrRemoteRequired
}

func xdgOpenPath(path string) (int, error) {
	cmd := exec.Command("xdg-open", path)
	err := cmd.Run()
	return cmd.ProcessState.ExitCode(), err
}

func (p Path) RemoteLaunch(options map[string]string) error {
	// empty if no option selected
	fzfKey := options[frontend.OptionFzfKey]

	// try opening the uri pointed to by to the shortcut
	path := string(p)

	// open the submitted path
	if fzfKey == "" {
		exitCode, err := xdgOpenPath(path)

		// 3,4 have workarounds, the rest are failures
		if exitCode != 3 && exitCode != 4 {
			return err
		}

		fzfKey = "ctrl-p"
	}

	// open parent
	if fzfKey == "ctrl-p" {
		parent := filepath.Dir(path)
		_, err := xdgOpenPath(parent)

		return err
	}

	// highlight in nautilus
	if fzfKey == "ctrl-n" {
		// open file in nautilus and highlight it
		return exec.Command("nautilus", path).Start()
	}

	// open in terminal
	if fzfKey == "ctrl-t" {
		// check if path is a file
		fileInfo, err := os.Stat(path)
		if err != nil {
			return err
		}

		if !fileInfo.IsDir() {
			// open parent instead
			path = filepath.Dir(path)
		}

		return exec.Command("x-terminal-emulator", "--working-directory", path).Start()
	}

	return ErrKeyNotHandled
}

// provide path on the disk
type PathProvider struct {
	PathProviderSettings
}

type PathProviderSettings struct {
	FdfindPath    string `json:"fdfind-path"`
	BaseDirectory string `json:"base-directory"`
	NoIgnoreVCS   bool   `json:"no-ignore-vcs"`
	HideFiles     bool   `json:"hide-files"`
	HideFolders   bool   `json:"hide-folders"`
}

func defaultPathProviderSettings() PathProviderSettings {
	return PathProviderSettings{
		FdfindPath:  "fdfind",
		NoIgnoreVCS: true,
		HideFiles:   false,
		HideFolders: false,
	}
}

func (p *PathProviderSettings) validate() (err error) {

	if p.FdfindPath == "" {
		p.FdfindPath = defaultPathProviderSettings().FdfindPath
	}

	if p.BaseDirectory == "" {
		p.BaseDirectory, err = utils.ResolvePath("~/")
		if err != nil {
			return fmt.Errorf("could not resolve base directory: %w", err)
		}
	}

	if !filepath.IsAbs(p.BaseDirectory) {
		return fmt.Errorf("base directory must be an absolute path")
	}

	return nil
}

func SetPathProviderSettings(conf *config.Config, settings PathProviderSettings) error {
	// get current settings
	currentSettings, err := utils.ValFromJSON[PathProviderSettings](conf.Providers[PathProviderKey])
	if err != nil {
		return err
	}

	// update settings
	currentSettings.FdfindPath = settings.FdfindPath
	currentSettings.BaseDirectory = settings.BaseDirectory
	currentSettings.NoIgnoreVCS = settings.NoIgnoreVCS
	currentSettings.HideFiles = settings.HideFiles
	currentSettings.HideFolders = settings.HideFolders

	currentSettings.validate()

	// save settings
	conf.Providers[PathProviderKey], err = utils.ValToJSON(currentSettings)
	if err != nil {
		return err
	}

	return conf.Save()
}

func NewPathProvider(conf *config.Config, options map[string]string) (EntryProvider, error) {

	// get current settings
	var settings PathProviderSettings
	settingsMap := conf.Providers[PathProviderKey]
	if len(settingsMap) == 0 {
		settings = defaultPathProviderSettings()
		err := SetPathProviderSettings(conf, settings)
		if err != nil {
			return nil, err
		}
	} else {
		err := utils.FromJSON(settingsMap, &settings)
		if err != nil {
			return nil, err
		}
	}

	provider := PathProvider{settings}

	if baseDirectory, ok := options[OptionBaseDirectory]; ok {
		provider.BaseDirectory = baseDirectory
	}

	if options[OptionIgnoreVCS] == "true" {
		provider.NoIgnoreVCS = false
	}

	if options[OptionHideFiles] == "true" {
		provider.HideFiles = true
	}

	if options[OptionHideFolders] == "true" {
		provider.HideFolders = true
	}

	if err := provider.validate(); err != nil {
		return nil, err
	}

	if provider.HideFiles && provider.HideFolders {
		return nil, fmt.Errorf("cannot hide both files and folders")
	}

	return provider, nil
}

func (p PathProvider) IsRemoteIndependent() bool {
	return false
}

func (p PathProvider) GetEntryReader() (io.Reader, error) {
	r, w := io.Pipe()

	args := []string{"--base-directory", p.BaseDirectory, "--relative-path", "--strip-cwd-prefix"}
	if p.NoIgnoreVCS {
		args = append(args, "--no-ignore-vcs")
	}
	if p.HideFiles {
		args = append(args, "--type", "d")
	}
	if p.HideFolders {
		args = append(args, "--type", "f")
	}

	fdfind := exec.Command(p.FdfindPath, args...)
	fdfind.Stdout = w

	err := fdfind.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		// no way to return the errors
		fdfind.Wait()
		w.Close()
	}()

	return r, nil
}

func (p PathProvider) Fetch(entry string) (Entry, bool) {
	// Join base directory with entry
	path := filepath.Join(p.BaseDirectory, entry)
	// Only accept if file exists and there are no errors
	if _, err := os.Stat(path); err == nil {
		return Path(path), true
	} else {
		return nil, false
	}
}
