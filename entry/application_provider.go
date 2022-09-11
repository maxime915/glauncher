package entry

import (
	"encoding/json"
	"fmt"
	"os/exec"

	config "github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/frontend"
)

const ApplicationProviderKey = "application-provider"

// TODO inspect the code source of Ubuntu dock https://github.com/micheleg/dash-to-dock to find more info

// identifier (e.g. "org.gnome.Calendar.desktop") for an application to launch
type Application struct {
	AppId     string  `json:"app-id"`
	PythonBin *string `json:"python-bin"`
}

func init() {
	RegisterEntryType[Application]()
	registerProvider(ApplicationProviderKey, NewApplicationProvider)
}

func (a Application) LaunchInFrontend(_ frontend.Frontend, _ map[string]string) error {
	return ErrRemoteRequired
}

func (a Application) RemoteLaunch(options map[string]string) error {
	cmd := exec.Command(
		*a.PythonBin,
		"-c",
		"from gi.repository import Gio; Gio.DesktopAppInfo.new('"+a.AppId+"').launch()",
	)
	return cmd.Run()
}

type ApplicationProvider = MapProvider[Application]

type applicationProviderSettings struct {
	PythonPath       string            `json:"python-path"`
	Blacklist        []string          `json:"id-black-list"`
	ExtraApplication map[string]string `json:"extra-app"`
}

func defaultApplicationSettings() applicationProviderSettings {
	return applicationProviderSettings{
		PythonPath:       "/usr/bin/python3",
		Blacklist:        nil,
		ExtraApplication: nil,
	}
}

func SetApplicationConfig(
	conf *config.Config,
	pythonPath string,
	blacklist []string,
	extraApplication map[string]string,
) error {

	// get current settings
	var currentSettings applicationProviderSettings
	settingsStr := conf.Providers[ApplicationProviderKey]
	if len(settingsStr) != 0 {
		err := json.Unmarshal([]byte(settingsStr), &currentSettings)
		if err != nil {
			return err
		}
	}

	// update settings
	currentSettings.PythonPath = pythonPath
	currentSettings.Blacklist = blacklist
	currentSettings.ExtraApplication = extraApplication

	// save settings
	settingsSerialized, err := json.Marshal(currentSettings)
	if err != nil {
		return err
	}

	conf.Providers[ApplicationProviderKey] = string(settingsSerialized)
	return conf.Save()
}

func NewApplicationProvider(conf *config.Config, options map[string]string) (EntryProvider, error) {
	// parse settings
	var settings applicationProviderSettings
	settingsStr := conf.Providers[ApplicationProviderKey]
	if len(settingsStr) == 0 {
		// get the defaults and store them
		settings = defaultApplicationSettings()
		settingsStr, err := json.Marshal(settings)
		if err != nil {
			return nil, err
		}

		conf.Providers[ApplicationProviderKey] = string(settingsStr)
		if err := conf.Save(); err != nil {
			return nil, err
		}
	} else {
		err := json.Unmarshal([]byte(settingsStr), &settings)
		if err != nil {
			return nil, err
		}
	}

	// not the same as the activity launcher... some activities (e.g. "Take a Screenshot") are not selected via DesktopAppInfo.get_all()
	data, err := exec.Command(
		settings.PythonPath,
		"-c",
		"from gi.repository import Gio; import json; "+
			"print(json.dumps([[app.get_id(), app.get_display_name()] "+
			"for app in Gio.DesktopAppInfo.get_all()]))",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("unable to get list of applications: %w", err)
	}

	var apps [][2]string
	err = json.Unmarshal(data, &apps)
	if err != nil {
		return nil, fmt.Errorf("unable to parse list of applications: %w", err)
	}

	blacklistSet := make(map[string]struct{})
	for _, blacklistItem := range settings.Blacklist {
		blacklistSet[blacklistItem] = struct{}{}
	}

	content := make(map[string]Application)
	for _, app := range apps {
		// skip items from the blacklist
		if _, ok := blacklistSet[app[0]]; ok {
			continue
		}

		content[app[1]] = Application{app[0], &settings.PythonPath}
	}

	for appName, appId := range settings.ExtraApplication {
		content[appName] = Application{appId, &settings.PythonPath}
	}

	return ApplicationProvider{
		Content: content,
		Prefix:  "🚀 ",
	}, nil
}
