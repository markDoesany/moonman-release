package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"release-launcher/backend/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:       "Release Launcher",
		Width:       1180,
		Height:      760,
		MinWidth:    900,
		MinHeight:   620,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup:   application.Startup,
		OnShutdown:  application.Shutdown,
		Bind:        []interface{}{application},
	})
	if err != nil {
		log.Fatal(err)
	}
}
