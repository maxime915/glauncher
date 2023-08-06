package frontend

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/maxime915/glauncher/config"
)

const (
	OptionFzfKey = "fzf-key"
	FzfKeyCTRL_T = "ctrl-t"
	FzfKeyCTRL_A = "ctrl-a"
	FzfKeyCTRL_P = "ctrl-p"
	FzfKeyCTRL_N = "ctrl-n"
	FzfKeyCTRL_D = "ctrl-d"
	FzfKeyCTRL_V = "ctrl-v"
)

var (
	ErrNoFrontendConfigured = errors.New("no frontend was configured")
	ErrNoEntrySelected      = errors.New("no entry selected")
	ErrNoNewLine            = errors.New("no newline found at the end of entry")
	ErrBadSelection         = errors.New("bad selection")
)

type Frontend interface {
	// StartFromReader starts the frontend and read entries from the reader
	StartFromReader(io.Reader, *config.Config) error
	// other option with channels ?
	// StartFromPipeline(chan <-string, ...) error

	// GetSelection waits for the input and return the selection
	GetSelection() (string, map[string]string, error)
	// other option with context for cancellation ?

	// AllowLocalExecution returns true if the frontend allows some entry to be
	// launched without a remote.
	AllowLocalExecution() bool
}

type FzfFrontend struct {
	cmd             *exec.Cmd
	selectionBuffer bytes.Buffer
}

func NewFzfFrontend() *FzfFrontend {
	return &FzfFrontend{}
}

func GetCtrlKeysOptions() []string {
	return []string{FzfKeyCTRL_T, FzfKeyCTRL_A, FzfKeyCTRL_P, FzfKeyCTRL_N, FzfKeyCTRL_D, FzfKeyCTRL_V}
}

func (f *FzfFrontend) StartFromReader(reader io.Reader, conf *config.Config) error {
	f.selectionBuffer = bytes.Buffer{}

	keys := GetCtrlKeysOptions()

	args := []string{
		"--multi",
	}

	for i, key := range keys {
		if strings.Contains(key, "\n") {
			return fmt.Errorf("keys[%d]=\"%v\" must not contain a newline", i, key)
		}
		action := fmt.Sprintf("%s:execute(echo %s)+accept", key, key)
		args = append(args, "--bind", action)
	}

	f.cmd = exec.Command(conf.FzfPath, args...)
	f.cmd.Stdin = reader
	f.cmd.Stdout = &f.selectionBuffer
	f.cmd.Stderr = os.Stderr

	return f.cmd.Start()
}

func (f *FzfFrontend) GetSelection() (string, map[string]string, error) {
	err := f.cmd.Wait()

	// if user presses ESC or CTRL-C, CTRL-D, ... fzf returns 130
	if f.cmd.ProcessState.ExitCode() == 130 {
		return "", nil, ErrNoEntrySelected
	}

	if err != nil {
		return "", nil, err
	}

	selectedBytes, err := io.ReadAll(&f.selectionBuffer)
	if err != nil {
		return "", nil, err
	}

	if len(selectedBytes) == 0 {
		return "", nil, ErrNoEntrySelected
	}

	// expect output="Entry\n" or output="Key\nEntry\n"
	output := string(selectedBytes)

	parts := strings.Split(output, "\n")
	if len(parts) < 2 {
		return "", nil, ErrNoNewLine
	}
	if len(parts) > 3 {
		return "", nil, ErrBadSelection
	}

	if len(parts) == 2 {
		return parts[0], map[string]string{}, nil
	}

	return parts[1], map[string]string{OptionFzfKey: parts[0]}, nil
}

func (f *FzfFrontend) AllowLocalExecution() bool {
	return true
}
