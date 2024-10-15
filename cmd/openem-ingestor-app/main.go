package main

import (
	"embed"
	"fmt"
	"log"

	core "github.com/SwissOpenEM/Ingestor/internal/core"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// String can be overwritten by using linker flags: -ldflags "-X main.version=VERSION"
var version string = "DEVELOPMENT_VERSION"

func main() {

	log.Printf("Version %s", version)

	if err := core.ReadConfig(core.DefaultConfigFileName()); err != nil {
		log.Print(fmt.Errorf("failed to read config file: %w", err))
	}

	config, err := core.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(core.GetFullConfig())
	log.Printf("Config file used: %s", core.GetCurrentConfigFilePath())

	// setup globus if we have a refresh token
	if config.Transfer.Globus.RefreshToken != "" {
		core.GlobusLoginWithRefreshToken(config.Transfer.Globus)
	}

	// Create an instance of the app structure
	app := NewApp(config, version)
	// Create application with options
	err = wails.Run(&options.App{
		Title:  "openem-ingestor",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
