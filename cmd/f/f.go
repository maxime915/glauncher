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

var (
	log  logger.Logger
	conf *config.Config
)

func init() {
	var err error
	conf, err = config.LoadConfig()
	if err != nil {
		logger.LoggerToStderr().Fatal(err)
	}

	if conf.LogFile == config.LogToStderr {
		log = logger.LoggerToStderr()
	} else {
		log, err = logger.LoggerToFile(conf.LogFile, false)

		if err != nil {
			logger.LoggerToStderr().Fatal(err)
		}
	}
}

func StartF(ctx *cli.Context) error {
	if ctx.NArg() > 1 {
		return cli.Exit("f takes at most 1 argument", 1)
	}

	// load the config for the util to work
	conf, err := config.LoadConfig()
	log.FatalIfErr(err)

	// get the entry handler while fzf is working
	var userRemote remote.Remote = nil
	if r, err := remote.GetRemote(conf); err == nil {
		userRemote = r
	} else {
		log.Print(err)
	}

	// read flags
	options := map[string]string{}
	for _, flag := range ctx.FlagNames() {
		options[flag] = ctx.String(flag)
	}

	// read args
	if ctx.NArg() == 1 {
		baseDirectory, err := filepath.Abs(ctx.Args().First())
		log.FatalIfErr(err)
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
		log.FatalIfErr(err)

		if userRemote != nil || provider.IsRemoteIndependent() {
			providers = append(providers, provider)
		}
	}

	// combine all entries
	var readerList []io.Reader
	for _, provider := range providers {
		reader, err := provider.GetEntryReader()
		log.FatalIfErr(err)

		readerList = append(readerList, reader)
	}
	reader := io.MultiReader(readerList...)

	fzf := frontend.NewFzfFrontend()
	err = fzf.StartFromReader(reader, conf)
	log.FatalIfErr(err)

	selected, newOptions, err := fzf.GetSelection()
	if err == frontend.ErrNoEntrySelected {
		return nil
	}
	log.FatalIfErr(err)

	// add options set by fzf
	for key, val := range newOptions {
		options[key] = val
	}

	entryHandled := false
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
			if userRemote == nil {
				continue
			}
			err = userRemote.HandleEntry(e, options)
		}

		log.FatalIfErr(err)
		entryHandled = true

		break
	}

	if !entryHandled {
		log.Fatalf("no provider could handle the selection: %s\n", selected)
	}

	if options["restart"] == "true" {
		return StartF(ctx)
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

	// StartF never returns an error so this is useless
	log.FatalIfErr(app.Run(os.Args))
}
