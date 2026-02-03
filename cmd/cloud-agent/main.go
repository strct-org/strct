package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/strct-org/strct-agent/internal/app"
	"github.com/strct-org/strct-agent/internal/config"
)

func main() {
	cfg := config.Load()
	
	agent := app.New(cfg)

	agent.Bootstrap()

	go agent.Start()

	waitForShutdown()
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down gracefully...")
}