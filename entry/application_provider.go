package entry

import (
	"encoding/json"
	"fmt"
	"os/exec"

	config "github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/frontend"
	"github.com/maxime915/glauncher/utils"
)

// Uniform identifier
//
// Deprecated: use DesktopFileProvider for fewer dependencies and issues
const ApplicationProviderKey = "application-provider"

// identifier (e.g. "org.gnome.Calendar.desktop") for an application to launch
//
// Deprecated: use DesktopFileProvider for fewer dependencies and issues
type Application struct {
	AppId     string  `json:"app-id"`
	PythonBin *string `json:"python-bin"`
}

func init() {
	RegisterEntryType[Application]()
	registerProvider(ApplicationProviderKey, NewApplicationProvider)
}

// Deprecated: use DesktopFileProvider for fewer dependencies and issues
func (a Application) LaunchInFrontend(_ frontend.Frontend, options map[string]string) error {
	if options[frontend.OptionFzfKey] != frontend.FzfKeyCTRL_D {
		return ErrRemoteRequired
	}

	// get the config
	conf, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// get application settings
	settings, err := utils.ValFromJSON[applicationProviderSettings](conf.Providers[ApplicationProviderKey])
	if err != nil {
		return err
	}

	// update it
	settings.Blacklist = append(settings.Blacklist, a.AppId)
	settingsSerialized, err := utils.ValToJSON(settings)
	if err != nil {
		return err
	}
	conf.Providers[ApplicationProviderKey] = settingsSerialized

	// commit
	err = conf.Save()
	if err != nil {
		return err
	}

	options["restart"] = "true"

	return nil
}

// Deprecated: use DesktopFileProvider for fewer dependencies and issues
func (a Application) RemoteLaunch(options map[string]string) error {
	cmd := exec.Command(
		*a.PythonBin,
		"-c",
		"from gi.repository import Gio; Gio.DesktopAppInfo.new('"+a.AppId+"').launch()",
	)
	return cmd.Run()
}

// Deprecated: use DesktopFileProvider for fewer dependencies and issues
type ApplicationProvider = MapProvider[Application]

// Deprecated: use DesktopFileProvider for fewer dependencies and issues
type applicationProviderSettings struct {
	PythonPath       string            `json:"python-path"`
	Blacklist        []string          `json:"application-id-blacklist"`
	ExtraApplication map[string]string `json:"application-extra"`
}

func defaultApplicationSettings() applicationProviderSettings {
	return applicationProviderSettings{
		PythonPath:       "/usr/bin/python3",
		Blacklist:        nil,
		ExtraApplication: nil,
	}
}

// Update the config the the ApplicationProvider
//
// Deprecated: use DesktopFileProvider for fewer dependencies and issues
func SetApplicationConfig(
	conf *config.Config,
	pythonPath string,
	blacklist []string,
	extraApplication map[string]string,
) error {

	// get current settings
	currentSettings, err := utils.ValFromJSON[applicationProviderSettings](conf.Providers[ApplicationProviderKey])
	if err != nil {
		return err
	}

	// update settings
	currentSettings.PythonPath = pythonPath
	currentSettings.Blacklist = blacklist
	currentSettings.ExtraApplication = extraApplication

	// save settings
	settingsSerialized, err := utils.ValToJSON(currentSettings)
	if err != nil {
		return err
	}

	conf.Providers[ApplicationProviderKey] = settingsSerialized
	return conf.Save()
}

// Build a new ApplicationProvider
//
// Deprecated: use DesktopFileProvider for fewer dependencies and issues
func NewApplicationProvider(conf *config.Config, options map[string]string) (EntryProvider, error) {
	// parse settings
	var settings applicationProviderSettings
	settingsMap := conf.Providers[ApplicationProviderKey]
	if len(settingsMap) == 0 {
		// get the defaults and store them
		settings = defaultApplicationSettings()
		err := SetApplicationConfig(conf, settings.PythonPath, settings.Blacklist, settings.ExtraApplication)
		if err != nil {
			return nil, err
		}
	} else {
		err := utils.FromJSON(settingsMap, &settings)
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
		Content:           content,
		Prefix:            "@ ",
		RemoteIndependent: true, // blacklisting apps is remote independent
	}, nil
}
