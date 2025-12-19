package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/bidirect/internal/udp"
)

func main() {
	port := flag.Int("port", 5555, "UDP port")
	flag.Parse()

	fmt.Printf("Starting UDP test server on 0.0.0.0:%d\n", *port)

	server := udp.NewServer(*port)
	if err := server.Start(); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("UDP server running. Press Ctrl+C to stop.")
	fmt.Println("Send images with: ./send-udp image.webp localhost:5555")

	// Mostrar frames recibidos
	go func() {
		rb := server.GetRingBuffer()
		for {
			if frame, ok := rb.ReadLatest(); ok && frame != nil {
				fmt.Printf("Frame in buffer: %dx%d (%d bytes)\n", frame.Width, frame.Height, len(frame.Data))
			}
		}
	}()

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nStopping server...")
	server.Stop()
}
