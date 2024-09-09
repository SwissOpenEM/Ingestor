package main

import (
	"fmt"
	"log"

	core "github.com/SwissOpenEM/Ingestor/internal/core"

	"github.com/spf13/viper"
)

func main() {
	if err := core.ReadConfig(); err != nil {
		panic(fmt.Errorf("Failed to read config file: %w", err))
	}
	log.Printf("Config file used: %s", viper.ConfigFileUsed())
	log.Println(viper.AllSettings())

}
