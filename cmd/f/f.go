package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/maxime915/glauncher/config"
	"github.com/maxime915/glauncher/entry"
	"github.com/maxime915/glauncher/frontend"
	"github.com/maxime915/glauncher/logger"
	"github.com/maxime915/glauncher/remote"
	"github.com/urfave/cli/v2"
)

func StartF(ctx *cli.Context) error {
	if ctx.NArg() > 1 {
		return cli.Exit("f takes at most 1 argument", 1)
	}

	// load the config for the util to work
	conf, err := config.LoadConfig()
	logger.FatalIfErr(err)

	// get the entry handler while fzf is working
	remote, err := remote.GetRemote(conf)
	logger.FatalIfErr(err)

	ctx.String("remote")

	// set sensible defaults
	options := map[string]string{
		entry.OptionBaseDirectory:       conf.BaseDirectory,
		entry.OptionOpenInTerminal:      "false",
		entry.OptionHighlightInNautilus: "false",
	}

	// read flags
	for _, flag := range ctx.FlagNames() {
		options[flag] = ctx.String(flag)
	}

	if ctx.NArg() == 1 {
		baseDirectory, err := filepath.Abs(ctx.Args().First())
		logger.FatalIfErr(err)
		options[entry.OptionBaseDirectory] = baseDirectory
	}

	// build blacklist set
	blacklistSet := make(map[string]struct{}, len(conf.Blacklist))
	for _, blacklistItem := range conf.Blacklist {
		blacklistSet[blacklistItem] = struct{}{}
	}

	// build all providers
	var providers []entry.EntryProvider
	for name, newProviderFun := range entry.GetRegisteredProviderFun() {
		if _, ok := blacklistSet[name]; ok {
			continue
		}

		provider, err := newProviderFun(conf, options)
		logger.FatalIfErr(err)
		providers = append(providers, provider)
	}

	// combine all entries
	var readerList []io.Reader
	for _, provider := range providers {
		reader, err := provider.GetEntryReader()
		logger.FatalIfErr(err)

		readerList = append(readerList, reader)
	}
	reader := io.MultiReader(readerList...)

	fzf := frontend.NewFzfFrontend()
	err = fzf.StartFromReader(reader, conf)
	logger.FatalIfErr(err)

	selected, newOptions, err := fzf.GetSelection()
	if err == frontend.ErrNoEntrySelected {
		return nil
	}
	logger.FatalIfErr(err)

	// add options set by fzf
	for key, val := range newOptions {
		options[key] = val
	}

	for _, provider := range providers {
		// fetch entry
		e, ok := provider.Fetch(selected)
		if !ok {
			continue
		}

		// launch entry

		// try from the frontend first
		err = nil
		if fzf.AllowLocalExecution() {
			err = e.LaunchInFrontend(fzf, options)
		}

		// fallback on the backend if necessary
		if err == entry.ErrRemoteRequired {
			err = remote.HandleEntry(e, options)
		}

		logger.FatalIfErr(err)

		break
	}

	return nil
}

func main() {
	app := &cli.App{
		Name: "f",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: entry.OptionHideFolders,
			},
			&cli.BoolFlag{
				Name: entry.OptionHideFiles,
			},
			&cli.BoolFlag{
				Name: entry.OptionIgnoreVCS,
			},
		},
		Action: StartF,
	}

	app.Run(os.Args)
}
