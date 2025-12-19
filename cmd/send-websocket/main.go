package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso:")
		fmt.Println("  send-websocket imagen.webp [ws://host:puerto/stream]")
		fmt.Println("  send-websocket video.webm [ws://host:puerto/stream] [fps]")
		fmt.Println("")
		fmt.Println("Ejemplos:")
		fmt.Println("  send-websocket test.webp ws://127.0.0.1:8080/stream")
		fmt.Println("  send-websocket video.webm ws://127.0.0.1:8080/stream 30")
		os.Exit(1)
	}

	filePath := os.Args[1]
	wsURL := "ws://127.0.0.1:8080/stream"
	if len(os.Args) > 2 {
		wsURL = os.Args[2]
	}

	fps := 30
	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%d", &fps)
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	if ext == ".webm" {
		sendVideoFrames(filePath, wsURL, fps)
	} else if ext == ".webp" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
		sendImage(filePath, wsURL)
	} else {
		fmt.Printf("Formato no soportado: %s\n", ext)
		os.Exit(1)
	}
}

func sendImage(imagePath, wsURL string) {
	fmt.Printf("[IMAGE] Archivo: %s\n", imagePath)
	data, err := os.ReadFile(imagePath)
	if err != nil {
		fmt.Printf("[ERROR] Lectura de imagen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[IMAGE] Tamaño: %d bytes\n", len(data))

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err != nil {
		fmt.Printf("[ERROR] Conexión WebSocket a %s: %v\n", wsURL, err)
		os.Exit(1)
	}
	defer ws.Close()
	fmt.Printf("[WS] ✓ Conectado a %s\n", wsURL)

	packet := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(packet[0:4], uint32(len(data)))
	copy(packet[4:], data)

	err = websocket.Message.Send(ws, packet)
	if err != nil {
		fmt.Printf("[ERROR] Envío: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[IMAGE] ✓ Enviado (%d bytes)\n", len(packet))
	fmt.Printf("[IMAGE] ✓ Completado a %s\n", wsURL)
}

func sendVideoFrames(videoPath, wsURL string, fps int) {
	tmpDir, err := os.MkdirTemp("", "bidirect-frames-")
	if err != nil {
		fmt.Printf("Error creando directorio temporal: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("[VIDEO] Archivo: %s\n", videoPath)
	fmt.Printf("[VIDEO] Directorio temporal: %s\n", tmpDir)
	fmt.Printf("[VIDEO] Extrayendo frames a %d fps...\n", fps)

	outputPattern := filepath.Join(tmpDir, "frame-%04d.webp")
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%d,scale=500:-1", fps),
		"-q:v", "90",
		outputPattern,
	)

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("[ERROR] ffmpeg falló: %v\n", err)
		fmt.Println("[ERROR] Asegúrate de que ffmpeg esté instalado")
		os.Exit(1)
	}

	fmt.Printf("[VIDEO] ✓ Frames extraídos exitosamente\n")

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err != nil {
		fmt.Printf("[ERROR] Conexión WebSocket fallida a %s: %v\n", wsURL, err)
		os.Exit(1)
	}
	defer ws.Close()
	fmt.Printf("[WS] ✓ Conectado a %s\n", wsURL)

	frameDelay := time.Duration(1000/fps) * time.Millisecond
	frameCount := 0

	for i := 1; i < 10000; i++ {
		framePath := filepath.Join(tmpDir, fmt.Sprintf("frame-%04d.webp", i))
		data, err := os.ReadFile(framePath)
		if err != nil {
			fmt.Printf("[VIDEO] Total de frames: %d\n", frameCount)
			break
		}

		packet := make([]byte, 4+len(data))
		binary.LittleEndian.PutUint32(packet[0:4], uint32(len(data)))
		copy(packet[4:], data)

		err = websocket.Message.Send(ws, packet)
		if err != nil {
			fmt.Printf("[ERROR] Frame %d: %v\n", i, err)
			os.Exit(1)
		}

		frameCount++
		fmt.Printf("[FRAME %d] ✓ Enviado (%d bytes)\n", i, len(data))
		time.Sleep(frameDelay)
	}

	fmt.Printf("\n[VIDEO] ✓ Completado: %d frames enviados a %s\n", frameCount, wsURL)
}
