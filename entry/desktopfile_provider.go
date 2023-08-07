package entry

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	config "github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/frontend"
	"github.com/maxime915/glauncher/utils"
	"golang.org/x/exp/maps"
)

const DesktopFileProviderKey = "desktopFile-provider"
const (
	typeApplication = "Application"
	typeLink        = "Link"
	typeDirectory   = "Directory"
)

var (
	header         = regexp.MustCompile(`(^|\n)\[.+\]\n`)
	mainHeader     = regexp.MustCompile(`(^|\n)\[Desktop Entry\]\n`)
	localizedEntry = regexp.MustCompile(`([A-Za-z0-9\-]+(\[[A-Za-z0-9\-]+\])?)\s*=\s*(.+)\r*`)
	separator      = []byte("=")
)

type DesktopFile struct {
	Name       string
	Identifier string
}

func init() {
	RegisterEntryType[DesktopFile]()
	registerProvider(DesktopFileProviderKey, NewDesktopFileProvider)
}

func (d DesktopFile) LaunchInFrontend(_ frontend.Frontend, options map[string]string) error {
	if options[frontend.OptionFzfKey] != frontend.FzfKeyCTRL_D {
		return ErrRemoteRequired
	}

	// get the config
	conf, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// get application settings
	settings, err := utils.ValFromJSON[dfProviderSettings](conf.Providers[DesktopFileProviderKey])
	if err != nil {
		return err
	}

	// update it
	settings.Blacklist = append(settings.Blacklist, d.Identifier)
	settingsSerialized, err := utils.ValToJSON(settings)
	if err != nil {
		return err
	}
	conf.Providers[DesktopFileProviderKey] = settingsSerialized

	// commit
	err = conf.Save()
	if err != nil {
		return err
	}

	options["restart"] = "true"

	return nil
}

func (d DesktopFile) RemoteLaunch(options map[string]string) error {
	return exec.Command("gtk-launch", d.Identifier).Run()
}

type DesktopFileProvider = MapProvider[DesktopFile]

type dfProviderSettings struct {
	Blacklist []string `json:"df-id-blacklist"`
}

func defaultDfSettings() dfProviderSettings {
	return dfProviderSettings{
		Blacklist: nil,
	}
}

func SetDfConfig(
	conf *config.Config,
	blacklist []string,
) error {

	// get current settings
	currentSettings, err := utils.ValFromJSON[dfProviderSettings](conf.Providers[DesktopFileProviderKey])
	if err != nil {
		return err
	}

	// update settings
	currentSettings.Blacklist = blacklist

	// save settings
	settingsSerialized, err := utils.ValToJSON(currentSettings)
	if err != nil {
		return err
	}

	conf.Providers[DesktopFileProviderKey] = settingsSerialized
	return conf.Save()
}

func NewDesktopFileProvider(conf *config.Config, options map[string]string) (EntryProvider, error) {
	// parse settings
	var settings dfProviderSettings
	settingsMap := conf.Providers[DesktopFileProviderKey]
	if len(settingsMap) == 0 {
		// get the defaults and store them
		settings = defaultDfSettings()
		err := SetDfConfig(conf, settings.Blacklist)
		if err != nil {
			return nil, err
		}
	} else {
		err := utils.FromJSON(settingsMap, &settings)
		if err != nil {
			return nil, err
		}
	}

	blacklistSet := make(map[string]struct{}, len(settings.Blacklist))
	if len(settings.Blacklist) == 0 {
		blacklistSet = nil
	}
	for _, item := range settings.Blacklist {
		blacklistSet[item] = struct{}{}
	}
	desktopFiles, err := ScanMulti(candidatesDirectories(), blacklistSet)

	if err != nil {
		return DesktopFileProvider{}, err
	}

	return DesktopFileProvider{
		Content:           desktopFiles,
		Prefix:            "@ ",
		RemoteIndependent: true, // blacklisting a Desktop File is remote independent
	}, nil
}

func parseBool(value string) bool {
	value = strings.TrimSpace(value)
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	panic("'" + value + "' is not a valid boolean representation")
}

func lstToSet(lst []string) map[string]struct{} {
	set := make(map[string]struct{}, len(lst))
	for _, item := range lst {
		set[item] = struct{}{}
	}
	return set
}

type desktopFileInfo map[string]string

// readKV the content of a desktop file. return an error if [DesktopEntry] isn't found
func readKV(content []byte) (desktopFileInfo, error) {
	// find [Desktop Entry]
	pos := mainHeader.FindIndex(content)
	if len(pos) == 0 {
		return nil, fmt.Errorf("could not find mandatory [Desktop Entry] header")
	}
	content = content[pos[1]:] // seek right after the main header

	// find the next header
	pos = header.FindIndex(content)
	if len(pos) > 0 {
		content = content[:pos[0]]
	}
	// if len(pos) == 0 there is no more header, we can parse all content[...]

	data := make(map[string]string, 10)
	// find all key-value pairs and add them to the map
	match_lst := localizedEntry.FindAll(content, -1)
	for _, match := range match_lst {
		sep_idx := bytes.Index(match, separator)
		if sep_idx == -1 {
			panic("expected to find separator in k-v match, must check regex")
		}

		data[string(match[:sep_idx])] = string(match[sep_idx+1:])
	}

	return data, nil
}

func Read(path string) (df DesktopFile, shouldShow bool, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return df, false, err
	}

	dfInfo, err := readKV(content)
	if err != nil {
		return df, false, err
	}

	if !dfInfo.Visible() {
		return df, false, nil
	}

	df.Name = dfInfo.Name()
	_, fName := filepath.Split(path)
	df.Identifier = fName

	return df, true, nil
}

func (d desktopFileInfo) getBool(key string, def bool) bool {
	value, present := d.Get(key)
	if !present {
		return def
	}

	return parseBool(value)
}

func (d desktopFileInfo) getList(key string) []string {
	list, ok := d.Get(key)
	if !ok {
		return nil
	}

	return strings.Split(list, ":")
}

/// Useful fields of the Desktop Entry specification
/// see https://specifications.freedesktop.org/desktop-entry-spec/desktop-entry-spec-latest.html

// Name of the application
func (d desktopFileInfo) Name() string {
	return d.mustGet("Name")
}

func (d desktopFileInfo) type_() string {
	return d.mustGet("Type")
}

func (d desktopFileInfo) noDisplay() bool {
	return d.getBool("NoDisplay", false)
}

func (d desktopFileInfo) hidden() bool {
	return d.getBool("Hidden", false)
}

func (d desktopFileInfo) onlyShowIn() []string {
	return d.getList("OnlyShowIn")
}

func (d desktopFileInfo) notShowIn() []string {
	return d.getList("NotShowIn")
}

/// Exported methods in addition to the specification (mostly shortcuts)

func (d desktopFileInfo) IsApplication() bool {
	return d.type_() == typeApplication
}

// Whether the desktop entry should be presented to the user or not
func (d desktopFileInfo) Visible() bool {
	// we only consider applications here
	if !d.IsApplication() {
		return false
	}

	// respect DF settings
	if d.noDisplay() || d.hidden() {
		return false
	}

	// evaluate OnlyShowIn and NotShowIn

	shouldShow, shouldHide := false, false

	// lookup XDG_CURRENT_DESKTOP
	desktopList := strings.Split(os.Getenv("XDG_CURRENT_DESKTOP"), ":")
	onlyShowIn := lstToSet(d.onlyShowIn())
	notShowIn := lstToSet(d.notShowIn())

	for _, desktop := range desktopList {
		if _, present := onlyShowIn[desktop]; present {
			shouldShow = true
			break
		}
		if _, present := notShowIn[desktop]; present {
			shouldHide = true
			break
		}
	}

	// default action
	if !shouldShow && !shouldHide {
		if len(onlyShowIn) > 0 {
			shouldHide = true
		} else {
			shouldShow = true
		}
	}

	// should never happen but who knows how the code might be altered
	xor_check := (shouldShow || shouldHide) && !(shouldShow && shouldHide)
	if !xor_check {
		panic("logical error in visibility computation")
	}

	return shouldShow
}

/// Generic utils functions

func (d desktopFileInfo) mustGet(key string) string {
	return map[string]string(d)[key]
}

func (d desktopFileInfo) Get(key string) (string, bool) {
	val, ok := map[string]string(d)[key]
	return val, ok
}

/// Searching for desktop files

// candidatesDirectories search for directories that may contain .desktop files
func candidatesDirectories() []string {
	var directories []string = nil

	// XDG_DATA_HOME
	data_home := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if data_home == "" {
		home_dir := strings.TrimSpace(os.Getenv("HOME"))
		if home_dir != "" {
			data_home = filepath.Join(home_dir, ".local/share")
		}
	}
	if data_home != "" {
		directories = append(directories, filepath.Join(data_home, "applications"))
	}

	// search in XDG_DATA_DIRS
	for _, data_dir := range strings.Split(os.Getenv("XDG_DATA_DIRS"), ":") {
		data_dir = strings.TrimSpace(data_dir)
		if data_dir != "" {
			directories = append(directories, filepath.Join(data_dir, "applications"))
		}
	}

	return directories
}

func walkDesktopFiles(done <-chan struct{}, directories []string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errs := make(chan error, 1)

	go func() {
		defer close(paths)

		for _, dir := range directories {
			file, err := os.Open(dir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				errs <- err
				return
			}

			fStats, err := file.Stat()
			if err != nil {
				errs <- err
				return
			}

			if !fStats.IsDir() {
				errs <- fmt.Errorf("not a directory")
				return
			}

			files, err := filepath.Glob(filepath.Clean(dir) + "/*.desktop")
			if err != nil {
				errs <- err
				return
			}

			for _, dFile := range files {
				paths <- dFile
			}
		}
		errs <- nil
	}()

	return paths, errs
}

type result struct {
	error
	DesktopFile
}

func digester(blacklist map[string]struct{}, done <-chan struct{}, paths <-chan string, c chan<- result) {
	for path := range paths {
		df, show, err := Read(path)

		if err == nil {
			_, blacklisted := blacklist[df.Identifier]
			if !show || blacklisted {
				continue
			}
		}

		select {
		case c <- result{err, df}:
		case <-done:
			return
		}
	}
}

func ScanDirectory(dir string, blacklist map[string]struct{}) (map[string]DesktopFile, error) {
	file, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]DesktopFile, 0), nil
		}
		return nil, err
	}

	fStats, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !fStats.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}

	dir = filepath.Clean(dir)

	files, err := filepath.Glob(dir + "/*.desktop")
	if err != nil {
		return nil, err
	}

	fileMap := make(map[string]DesktopFile, len(files))
	for _, file := range files {
		df, show, err := Read(file)
		if err != nil {
			return nil, err
		}
		_, blacklisted := blacklist[df.Identifier]
		if show && !blacklisted {
			fileMap[df.Name] = df
		}
	}

	return fileMap, nil
}

func sendAndMerge(resChan chan map[string]DesktopFile, value map[string]DesktopFile, cancelled chan struct{}) {
	select {
	case resChan <- value:
		// OK, nothing else to do
	case other := <-resChan:
		// merge and send again
		maps.Copy(value, other)
		sendAndMerge(resChan, value, cancelled)
	case <-cancelled:
		// cancelled: discard any results
		return
	}
}

func scanMultipleMT(directories []string, blacklist map[string]struct{}) (map[string]DesktopFile, error) {
	// avoid a dead-lock if there are no directories to scan
	if len(directories) == 0 {
		return make(map[string]DesktopFile), nil
	}

	errChan := make(chan error, len(directories))
	resChan := make(chan map[string]DesktopFile, 1)
	done := make(chan struct{}, len(directories))
	cancelled := make(chan struct{})

	for _, dir := range directories {
		go func(path string) {
			dFiles, err := ScanDirectory(path, blacklist)
			if err != nil {
				errChan <- err
			} else {
				sendAndMerge(resChan, dFiles, cancelled)
				done <- struct{}{}
			}
		}(dir)
	}

	waiting := len(directories)
	for waiting > 0 {
		select {
		case <-done:
			waiting -= 1
		case err := <-errChan:
			close(cancelled)
			return nil, err
		}
	}

	return <-resChan, nil
}

func scanMultipleST(directories []string, blacklist map[string]struct{}) (map[string]DesktopFile, error) {
	results := make(map[string]DesktopFile, 32*len(directories))

	for _, dir := range directories {
		dFiles, err := ScanDirectory(dir, blacklist)
		if err != nil {
			return nil, err
		}
		maps.Copy(results, dFiles)
	}

	return results, nil
}

func scanMultiplePL(dirs []string, blacklist map[string]struct{}) (map[string]DesktopFile, error) {
	done := make(chan struct{})
	defer close(done)

	paths, errC := walkDesktopFiles(done, dirs)

	c := make(chan result)
	var wg sync.WaitGroup
	const numDigesters = 32
	wg.Add(numDigesters)
	for i := 0; i < numDigesters; i++ {
		go func() {
			digester(blacklist, done, paths, c)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c)
	}()

	resMap := make(map[string]DesktopFile, 128)
	for res := range c {
		if res.error != nil {
			return nil, res.error
		}
		resMap[res.DesktopFile.Name] = res.DesktopFile
	}

	if err := <-errC; err != nil {
		return nil, err
	}

	return resMap, nil
}

func ScanMulti(directories []string, blacklist map[string]struct{}) (map[string]DesktopFile, error) {
	return scanMultiplePL(directories, blacklist)
}
