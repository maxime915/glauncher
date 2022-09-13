package entry

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	config "github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/frontend"
)

const DesktopFileProviderKey = "desktopFile-provider"

type DesktopFile string

func init() {
	RegisterEntryType[DesktopFile]()
	registerProvider(DesktopFileProviderKey, NewDesktopFileProvider)
}

func (d DesktopFile) LaunchInFrontend(_ frontend.Frontend, _ map[string]string) error {
	return ErrRemoteRequired
}

func (d DesktopFile) RemoteLaunch(options map[string]string) error {
	return exec.Command("gio", "launch", string(d)).Run()
}

// or maybe a custom struct with async fdfind ? that becomes harder...
type DesktopFileProvider = MapProvider[DesktopFile]

func NewDesktopFileProvider(conf *config.Config, options map[string]string) (EntryProvider, error) {
	prefix := "XDG_DATA_DIRS="

	var folders []string
	for _, keyVar := range os.Environ() {

		if strings.HasPrefix(keyVar, prefix) {
			folders = strings.Split(strings.TrimPrefix(keyVar, prefix), ":")
			break
		}
	}

	desktopFiles := make(map[string]DesktopFile)

	for _, folder := range folders {
		if _, err := os.Stat(folder); err != nil {
			continue
		}

		appFolder := filepath.Join(folder, "applications")
		if _, err := os.Stat(appFolder); err != nil {
			continue
		}

		// list all content
		files, err := os.ReadDir(appFolder)
		if err != nil {
			return nil, err
		}

		suffix := ".desktop"
		for _, file := range files {
			fName := file.Name()
			fPath := filepath.Join(appFolder, fName)
			if strings.HasSuffix(fName, suffix) {
				appName := strings.TrimSuffix(fName, suffix)
				if fName == appName || len(appName) == 0 {
					continue
				}

				desktopFiles[appName] = DesktopFile(fPath)
			}
		}
	}

	return DesktopFileProvider{
		Content: desktopFiles,
		Prefix:  "# ",
	}, nil
}
