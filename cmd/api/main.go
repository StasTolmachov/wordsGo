package main

import (
	"log"

	"wordsGo/internal/config"
	"wordsGo/internal/server"
	"wordsGo/slogger"
)

// @title User Management
// @version 1.0
// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.basic BasicAuth
func main() {
	slogger.MakeLogger(true)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration load failed: %s", err)
	}
	slogger.Log.Debug("Config loaded", "config:", cfg)

	server.Run(*cfg)

}
