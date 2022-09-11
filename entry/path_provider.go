package entry

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/frontend"
)

const (
	PathProviderKey           = "path-provider"
	OptionBaseDirectory       = "base-directory"
	OptionHideFiles           = "hide-files"
	OptionHideFolders         = "hide-folders"
	OptionIgnoreVCS           = "no-ignore-vcs"
	OptionOpenInTerminal      = "open-in-terminal"
	OptionHighlightInNautilus = "highlight-in-nautilus"
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

		return exec.Command("x-terminal-emulator", "--working-directory", path).Run()
	}

	return ErrKeyNotHandled
}

// provide path on the disk
type PathProvider struct {
	fdfindPath    string
	baseDirectory string
	noIgnoreVCS   bool
	hideFiles     bool
	hideFolders   bool
}

func NewPathProvider(conf *config.Config, options map[string]string) (EntryProvider, error) {
	provider := PathProvider{
		fdfindPath:  conf.FdfindPath,
		noIgnoreVCS: true,
	}

	var ok bool
	provider.baseDirectory, ok = options[OptionBaseDirectory]
	if !ok {
		provider.baseDirectory = conf.BaseDirectory
	}

	if options[OptionIgnoreVCS] == "true" {
		provider.noIgnoreVCS = false
	}

	if options[OptionHideFiles] == "true" {
		provider.hideFiles = true
	}

	if options[OptionHideFolders] == "true" {
		provider.hideFolders = true
	}

	if provider.hideFiles && provider.hideFolders {
		return nil, fmt.Errorf("cannot hide both files and folders")
	}

	return provider, nil
}

func (p PathProvider) GetEntryReader() (io.Reader, error) {
	r, w := io.Pipe()

	args := []string{"--base-directory", p.baseDirectory, "--relative-path", "--strip-cwd-prefix"}
	if p.noIgnoreVCS {
		args = append(args, "--no-ignore-vcs")
	}
	if p.hideFiles {
		args = append(args, "--type", "d")
	}
	if p.hideFolders {
		args = append(args, "--type", "f")
	}

	fdfind := exec.Command(p.fdfindPath, args...)
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
	path := filepath.Join(p.baseDirectory, entry)
	// Only accept if file exists and there are no errors
	if _, err := os.Stat(path); err == nil {
		return Path(path), true
	} else {
		return nil, false
	}
}
