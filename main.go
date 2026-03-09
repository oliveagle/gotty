package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/oliveagle/gotty/backend/localcommand"
	"github.com/oliveagle/gotty/backend/zellijcommand"
	"github.com/oliveagle/gotty/pkg/homedir"
	"github.com/oliveagle/gotty/server"
	"github.com/oliveagle/gotty/utils"
)

func main() {
	app := &cli.App{
		Name:    "gotty",
		Version: Version + "+" + CommitID,
		Usage:    "Share your terminal as a web application",
	}
	cli.AppHelpTemplate = helpTemplate

	appOptions := &server.Options{}
	if err := utils.ApplyDefaultValues(appOptions); err != nil {
		exit(err, 1)
	}
	backendOptions := &localcommand.Options{}
	if err := utils.ApplyDefaultValues(backendOptions); err != nil {
		exit(err, 1)
	}

	cliFlags, flagMappings, err := utils.GenerateFlags(appOptions, backendOptions)
	if err != nil {
		exit(err, 3)
	}

	app.Flags = append(
		cliFlags,
		&cli.StringFlag{
			Name:    "config",
			Value:   "~/.config/gotty/config.jsonc",
			Usage:   "Config file path (.jsonc, .json, or HCL format)",
			EnvVars: []string{"GOTTY_CONFIG"},
		},
	)

	app.Action = func(c *cli.Context) error {
		args := c.Args().Slice()
		if len(args) == 0 {
			args = []string{"bash"}
		}

		configFile := c.String("config")
		// Check if config file exists, try alternative paths if default doesn't exist
		configPaths := []string{
			configFile,
			"~/.config/gotty/config.jsonc",
			"~/.config/gotty/config.json",
			"~/.gotty", // Legacy HCL format
		}

		for _, path := range configPaths {
			expandedPath := homedir.Expand(path)
			if _, err := os.Stat(expandedPath); err == nil {
				configFile = path
				break
			}
		}

		if _, err := os.Stat(homedir.Expand(configFile)); err == nil {
			if err := utils.ApplyConfigFile(configFile, appOptions, backendOptions); err != nil {
				exit(err, 2)
			}
		}

		utils.ApplyFlags(cliFlags, flagMappings, c, appOptions, backendOptions)

		appOptions.EnableTLSClientAuth = c.IsSet("tls-ca-crt")

		// Track if permit-write was explicitly set by user
		if c.IsSet("permit-write") {
			appOptions.SetPermitWriteExplicit()
		}

		// Allow backend to be overridden by command line
		backendType := appOptions.Backend
		if c.IsSet("backend") {
			backendType = c.String("backend")
		}

		err = appOptions.Validate()
		if err != nil {
			exit(err, 6)
		}

		// Create factory based on backend type
		var factory server.Factory
		switch backendType {
		case "zellij":
			zellijOptions := &zellijcommand.Options{}
			if err := utils.ApplyDefaultValues(zellijOptions); err != nil {
				exit(err, 1)
			}
			factory, err = zellijcommand.NewFactory(args[0], args[1:], zellijOptions)
			if err != nil {
				exit(err, 3)
			}
			log.Printf("Using zellij backend for persistent sessions")
		default:
			factory, err = localcommand.NewFactory(args[0], args[1:], backendOptions)
			if err != nil {
				exit(err, 3)
			}
		}

		hostname, _ := os.Hostname()
		appOptions.TitleVariables = map[string]interface{}{
			"command":  args[0],
			"argv":     args[1:],
			"hostname": hostname,
		}

		// Set build info for server
		server.BuildVersion = Version
		server.BuildCommit = CommitID
		server.BuildTime = BuildTime

		srv, err := server.New(factory, appOptions)
		if err != nil {
			exit(err, 3)
		}

		ctx, cancel := context.WithCancel(context.Background())
		gCtx, gCancel := context.WithCancel(context.Background())

		log.Printf("GoTTY is starting with command: %s", strings.Join(args, " "))

		errs := make(chan error, 1)
		go func() {
			errs <- srv.Run(ctx, server.WithGracefullContext(gCtx))
		}()
		err = waitSignals(errs, cancel, gCancel)

		if err != nil && err != context.Canceled {
			fmt.Printf("Error: %s\n", err)
			exit(err, 8)
		}

		return nil
	}
	if err := app.Run(os.Args); err != nil {
		exit(err, 1)
	}
}

func exit(err error, code int) {
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(code)
}

func waitSignals(errs chan error, cancel context.CancelFunc, gracefullCancel context.CancelFunc) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(
		sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	select {
	case err := <-errs:
		return err

	case s := <-sigChan:
		switch s {
		case syscall.SIGINT:
			gracefullCancel()
			fmt.Println("C-C to force close")
			select {
			case err := <-errs:
				return err
			case <-sigChan:
				fmt.Println("Force closing...")
				cancel()
				return <-errs
			}
		default:
			cancel()
			return <-errs
		}
	}
}
