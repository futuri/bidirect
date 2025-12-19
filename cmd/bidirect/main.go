package main

import (
	"flag"
	"os"

	"github.com/example/bidirect/internal/config"
	"github.com/example/bidirect/internal/logging"
	"github.com/example/bidirect/internal/window"
)

func main() {
	cfg := config.DefaultConfig()

	flag.IntVar(&cfg.InitialSize, "size", cfg.InitialSize, "Initial window size")
	flag.IntVar(&cfg.WSPort, "port", cfg.WSPort, "WebSocket port")
	flag.Parse()

	logging.Infof("Starting BiDirect - WebSocket streaming receiver on port %d", cfg.WSPort)

	w, err := window.NewWindow(cfg)
	if err != nil {
		logging.Errorf("Failed to create window: %v", err)
		os.Exit(1)
	}

	if err := w.Run(); err != nil {
		logging.Errorf("Window error: %v", err)
		os.Exit(1)
	}
}
