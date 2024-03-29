package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/entry"
	"github.com/maxime915/glauncher/logger"
	"github.com/maxime915/glauncher/remote"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func loadConfig() (logger.Logger, *config.Config) {
	conf, err := config.LoadConfig()
	if err != nil {
		logger.LoggerToStderr().Fatal(err)
	}

	if conf.LogFile == config.LogToStderr {
		return logger.LoggerToStderr(), conf
	} else {
		log, err := logger.LoggerToFile(conf.LogFile, false)

		if err != nil {
			logger.LoggerToStderr().Fatal(err)
		}
		return log, conf
	}
}

func getRemote(flag string) (remote.Remote, error) {
	_, conf := loadConfig()

	switch flag {
	case remote.RemoteHTTP:
		httpConfig, err := remote.GetHTTPConfig(conf)
		if err != nil {
			return nil, err
		}

		return remote.NewHTTPConnection(httpConfig), nil
	case remote.RemoteRPC:
		rpcConfig, err := remote.GetRPCConfig(conf)
		if err != nil {
			return nil, err
		}

		return remote.NewRPCConnection(rpcConfig)
	case "":
		return remote.GetRemoteAndConnect(conf, false)
	default:
		return nil, remote.ErrInvalidRemote
	}
}

// KillRemote : stop remote (either default or specified)
func KillRemote(ctx *cli.Context) error {
	log, _ := loadConfig()

	if ctx.NArg() > 0 {
		return cli.Exit("too many arguments", 1)
	}

	remote, err := getRemote(ctx.String("remote"))
	log.FatalIfErr(err)

	return remote.Close()
}

// StartRemote : start remote (either default or specified)
func StartRemote(cliCtx *cli.Context) error {
	if cliCtx.NArg() > 0 {
		return cli.Exit("too many arguments", 1)
	}

	log, _ := loadConfig()

	remote, err := getRemote(cliCtx.String("remote"))
	log.FatalIfErr(err)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	interrupted := true
	defer stop()

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		err := remote.Start()
		interrupted = false
		stop()
		return err
	})
	errGroup.Go(func() error {
		<-ctx.Done()
		if interrupted {
			return remote.Close()
		}
		return nil
	})

	return errGroup.Wait()
}

// SaveEntryProviderSettings : setup the config file from this function
func SaveEntryProviderSettings(ctx *cli.Context) error {
	log, conf := loadConfig()

	if ctx.NArg() > 0 {
		return cli.Exit("too many arguments", 1)
	}

	err := entry.AddShortcutsToConfig(conf, map[string]entry.ShortCut{
		"notion":      "https://www.notion.so/",
		"overleaf":    "https://www.overleaf.com/project",
		"slides":      "https://docs.google.com/presentation/u/0/",
		"ddg":         "https://duckduckgo.com",
		"ggl":         "https://www.google.com/",
		"myuliege":    "https://my.uliege.be/FW/index.do",
		"ecampus":     "https://ecampus.uliege.be/",
		"desmos":      "https://www.desmos.com/calculator",
		"keep":        "https://keep.google.com",
		"doi 2 bib":   "https://www.doi2bib.org",
		"aol":         "https://mail.aol.com/webmail-std/en-us/suite",
		"gmail":       "https://mail.google.com/mail/u/0/",
		"calendar":    "https://calendar.google.com/calendar/u/0/",
		"unif mail":   "https://mail.ulg.ac.be",
		".zshrc":      "~/.zshrc",
		".fdignore":   "~/.fdignore",
		"oamg":        "https://oa.mg",
		"apple-music": "https://music.apple.com/be/browse",
	}, false)
	log.FatalIfErr(err)

	err = entry.AddCommandsToConfig(conf, map[string]entry.Command{
		"open-notebook": {
			Name: "/home/maxime/miniconda3/envs/ml_env/bin/jupyter",
			Args: []string{
				"notebook",
				"--notebook-dir=/home/maxime/notebooks",
			},
			SecondDelay:    0,
			CloseOnFailure: false,
		},
		"start-notebook": {
			Name: "/home/maxime/miniconda3/envs/ml_env/bin/jupyter",
			Args: []string{
				"notebook",
				"--notebook-dir=/home/maxime/notebooks",
			},
			SecondDelay:    0,
			CloseOnFailure: false,
		},
		"waker": {
			Name: "/home/maxime/go/bin/waker",
			Args: []string{
				"remote-http",
				"-n",
				"waker.maximeamodei.be",
			},
			SecondDelay:    0,
			CloseOnFailure: false,
		},
		"ipython": {
			Name: "/home/maxime/miniconda3/envs/ml_env/bin/python",
			Args: []string{
				"-c",
				"from math import *; import numpy as np; from IPython import embed; embed(colors='neutral')",
			},
			SecondDelay:    0,
			CloseOnFailure: true,
		},
		"ssh-dtop2": {
			Name: "/usr/bin/ssh",
			Args: []string{
				"-o",
				"ProxyCommand=/usr/bin/cloudflared access ssh --hostname ssh.maximeamodei.be",
				"-i",
				"/home/maxime/.ssh/id_rsa",
				"maximw@localhost-dtop2",
			},
			SecondDelay:    0,
			CloseOnFailure: false,
		},
		"t": {
			Name:           "/usr/bin/zsh",
			Args:           nil,
			SecondDelay:    0,
			CloseOnFailure: false,
		},
		"ping": {
			Name: "/usr/bin/ping",
			Args: []string{
				"-i",
				"0.2",
				"1.1",
				"-c",
				"5",
			},
			SecondDelay:    3,
			CloseOnFailure: true,
		},
		"lock": {
			Name:           "/usr/bin/xdg-screensaver",
			Args:           []string{"lock"},
			SecondDelay:    0,
			CloseOnFailure: false,
		},
	}, false)
	log.FatalIfErr(err)

	return nil
}

func main() {
	log, _ := loadConfig()

	app := &cli.App{
		Name: "glauncher",
		Commands: []*cli.Command{
			{
				Name:    "kill-remote",
				Aliases: []string{"k"},
				Usage:   "kill a running remote",
				Action:  KillRemote,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "remote",
						Aliases: []string{"r"},
						Usage:   "`REMOTE` to kill",
					},
				},
			},
			{
				Name:    "start-remote",
				Aliases: []string{"s"},
				Usage:   "start a remote",
				Action:  StartRemote,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "remote",
						Aliases: []string{"r"},
						Usage:   "`REMOTE` to kill",
					},
				},
			},
			{
				Name:   "save-entry-provider-settings",
				Usage:  "Save settings for the entry provider",
				Action: SaveEntryProviderSettings,
			},
		},
	}

	log.FatalIfErr(app.Run(os.Args))
}
