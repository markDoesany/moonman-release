package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"release-launcher/backend/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	openGUI, err := resolveCommand(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Moonman Release:", err)
		fmt.Fprintln(os.Stderr, "Usage: moonman-release [open]")
		os.Exit(2)
	}
	if !openGUI {
		printUsage()
		return
	}

	application, err := app.New()
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:       "Moonman Release",
		Width:       1180,
		Height:      760,
		MinWidth:    900,
		MinHeight:   620,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup:   application.Startup,
		OnShutdown:  application.Shutdown,
		OnBeforeClose: func(_ context.Context) bool {
			return application.BuildRunning()
		},
		Bind: []interface{}{application},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func resolveCommand(args []string) (bool, error) {
	if len(args) == 0 || (len(args) == 1 && args[0] == "open") {
		return true, nil
	}
	if len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		return false, nil
	}
	if len(args) == 0 {
		return false, fmt.Errorf("unknown command")
	}
	return false, fmt.Errorf("unknown command %q", args[0])
}

func printUsage() {
	fmt.Println("Moonman Release")
	fmt.Println("Usage: moonman-release [open]")
	fmt.Println()
	fmt.Println("  open    Open the Moonman Release desktop application")
}
