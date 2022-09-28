package entry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	config "github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/frontend"
	"github.com/maxime915/glauncher/utils"
)

const CommandProviderKey = "command-provider"

var ErrUnableToRemoteLaunchCommand = errors.New("unable to RemoteLaunch() this command")

func init() {
	RegisterEntryType[Command]()
	registerProvider(CommandProviderKey, NewCommandProvider)
}

// command to run in the current terminal
type Command struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
	// how much should we wait before exiting after a successful run
	SecondDelay int `json:"second_delay"`
	// failure leave the window open by default
	CloseOnFailure bool `json:"close_on_failure"`
}

func (c Command) LaunchInFrontend(_ frontend.Frontend, _ map[string]string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		if c.CloseOnFailure {
			return nil
		}

		// wait for ctrl-c
		<-ctx.Done()
		return err
	} else {
		// either timeout or ctrl-c
		select {
		case <-time.After(time.Duration(c.SecondDelay) * time.Second):
		case <-ctx.Done():
		}
	}
	return nil
}

func (c Command) RemoteLaunch(options map[string]string) error {
	return ErrUnableToRemoteLaunchCommand
}

// provide commands
type CommandProvider = MapProvider[Command]

// struct to store the commands in the config files
type commandSettings struct {
	CommandList map[string]Command `json:"commands-list"`
	Prefix      string             `json:"prefix"`
}

func defaultCommandList() commandSettings {
	return commandSettings{map[string]Command{}, "< "}
}

func AddCommandsToConfig(conf *config.Config, commands map[string]Command, override bool) error {
	// get current commands
	currentCommands, err := utils.ValFromJSON[commandSettings](conf.Providers[CommandProviderKey])
	if err != nil {
		return nil
	}

	if currentCommands.CommandList == nil {
		currentCommands.CommandList = make(map[string]Command, len(commands))
	}

	// check for overriding
	if !override {
		var duplicates []string
		// check for duplicates
		for k := range commands {
			if _, ok := currentCommands.CommandList[k]; ok {
				duplicates = append(duplicates, k)
			}
		}

		if len(duplicates) > 0 {
			return fmt.Errorf("duplicate commands: %s", duplicates)
		}
	}

	// merge commands
	for k, v := range commands {
		currentCommands.CommandList[k] = v
	}

	// save commands
	commandsSerialized, err := utils.ValToJSON(currentCommands)
	if err != nil {
		return err
	}

	conf.Providers[CommandProviderKey] = commandsSerialized
	return conf.Save()
}

func NewCommandProvider(conf *config.Config, options map[string]string) (EntryProvider, error) {
	// parse commands
	var commands commandSettings
	commandsMap := conf.Providers[CommandProviderKey]
	if len(commandsMap) == 0 {
		// get the defaults, and store them
		commands = defaultCommandList()
		err := AddCommandsToConfig(conf, commands.CommandList, false)
		if err != nil {
			return nil, err
		}
	} else {
		err := utils.FromJSON(commandsMap, &commands)
		if err != nil {
			return nil, err
		}
	}

	return CommandProvider{
		Content:           commands.CommandList,
		Prefix:            commands.Prefix,
		RemoteIndependent: true,
	}, nil
}
