package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso:")
		fmt.Println("  send-udp imagen.webp [host:puerto]")
		fmt.Println("  send-udp video.webm [host:puerto] [fps]")
		fmt.Println("")
		fmt.Println("Ejemplos:")
		fmt.Println("  send-udp test.webp 127.0.0.1:5555")
		fmt.Println("  send-udp video.webm 127.0.0.1:5555 30")
		os.Exit(1)
	}

	filePath := os.Args[1]
	addr := "127.0.0.1:5555"
	if len(os.Args) > 2 {
		addr = os.Args[2]
	}

	fps := 30
	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%d", &fps)
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	if ext == ".webm" {
		sendVideoFrames(filePath, addr, fps)
	} else if ext == ".webp" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
		sendImage(filePath, addr)
	} else {
		fmt.Printf("Formato no soportado: %s\n", ext)
		os.Exit(1)
	}
}

func sendImage(imagePath, addr string) {
	fmt.Printf("[IMAGE] Archivo: %s\n", imagePath)
	data, err := os.ReadFile(imagePath)
	if err != nil {
		fmt.Printf("[ERROR] Lectura de imagen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[IMAGE] Tamaño: %d bytes\n", len(data))

	conn, err := net.Dial("udp", addr)
	if err != nil {
		fmt.Printf("[ERROR] Conexión UDP a %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("[UDP] ✓ Conectado a %s\n", addr)

	packet := make([]byte, 5+len(data))
	packet[0] = 2
	binary.LittleEndian.PutUint32(packet[1:5], uint32(len(data)))
	copy(packet[5:], data)

	if len(packet) > 65535 {
		fmt.Printf("[IMAGE] Grande - fragmentando\n")
		sendFragmented(conn, data)
	} else {
		n, err := conn.Write(packet)
		if err != nil {
			fmt.Printf("[ERROR] Envío: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[IMAGE] ✓ Enviado (%d bytes)\n", n)
	}

	fmt.Printf("[IMAGE] ✓ Completado a %s\n", addr)
}

func sendVideoFrames(videoPath, addr string, fps int) {
	// Crear directorio temporal
	tmpDir, err := os.MkdirTemp("", "bidirect-frames-")
	if err != nil {
		fmt.Printf("Error creando directorio temporal: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("[VIDEO] Archivo: %s\n", videoPath)
	fmt.Printf("[VIDEO] Directorio temporal: %s\n", tmpDir)
	fmt.Printf("[VIDEO] Extrayendo frames a %d fps, redimensionando a 500px ancho...\n", fps)

	// Usar ffmpeg para extraer frames como WebP
	outputPattern := filepath.Join(tmpDir, "frame-%04d.webp")
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%d,scale=500:-1", fps), // Redimensionar a 500px ancho
		"-q:v", "90", // Calidad WebP (1-100, más bajo = mejor calidad)
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

	// Leer frames y enviar
	conn, err := net.Dial("udp", addr)
	if err != nil {
		fmt.Printf("[ERROR] Conexión UDP fallida a %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("[UDP] ✓ Conectado a %s\n", addr)

	frameDelay := time.Duration(1000/fps) * time.Millisecond
	frameCount := 0

	// Enviar cada frame
	for i := 1; i < 10000; i++ {
		framePath := filepath.Join(tmpDir, fmt.Sprintf("frame-%04d.webp", i))
		data, err := os.ReadFile(framePath)
		if err != nil {
			fmt.Printf("[VIDEO] Total de frames: %d\n", frameCount)
			break // No hay más frames
		}

		packet := make([]byte, 5+len(data))
		packet[0] = 2
		binary.LittleEndian.PutUint32(packet[1:5], uint32(len(data)))
		copy(packet[5:], data)

		if len(packet) > 65535 {
			fmt.Printf("[FRAME %d] Grande (%d bytes) - fragmentando...\n", i, len(data))
			sendFragmented(conn, data)
		} else {
			_, err = conn.Write(packet)
			if err != nil {
				fmt.Printf("[ERROR] Frame %d: %v\n", i, err)
				os.Exit(1)
			}
		}

		frameCount++
		fmt.Printf("[FRAME %d] ✓ Enviado (%d bytes)\n", i, len(data))
		time.Sleep(frameDelay)
	}

	fmt.Printf("\n[VIDEO] ✓ Completado: %d frames enviados a %s\n", frameCount, addr)
}

func sendFragmented(conn net.Conn, data []byte) {
	const maxPayload = 1200 // Seguro para UDP sin fragmentación IP
	fmt.Printf("[FRAGMENTED] Tamaño total: %d bytes, payload máximo: %d\n", len(data), maxPayload)

	// Primer paquete: tipo 0 + tamaño total + datos
	firstPacket := make([]byte, 5+maxPayload)
	firstPacket[0] = 0 // Tipo: inicio frame
	binary.LittleEndian.PutUint32(firstPacket[1:5], uint32(len(data)))

	remaining := data
	payloadSize := min(len(remaining), maxPayload)
	copy(firstPacket[5:], remaining[:payloadSize])
	n, err := conn.Write(firstPacket[:5+payloadSize])
	if err != nil {
		fmt.Printf("[FRAGMENTED] Error paquete inicio: %v\n", err)
		return
	}
	fmt.Printf("[FRAGMENTED] Paquete inicio: %d bytes enviados\n", n)
	remaining = remaining[payloadSize:]

	// Paquetes siguientes: tipo 1 + datos
	pktNum := 1
	for len(remaining) > 0 {
		payloadSize = min(len(remaining), maxPayload)
		packet := make([]byte, 1+payloadSize)
		packet[0] = 1 // Tipo: continuación
		copy(packet[1:], remaining[:payloadSize])
		n, err := conn.Write(packet)
		if err != nil {
			fmt.Printf("[FRAGMENTED] Error paquete %d: %v\n", pktNum, err)
			return
		}
		fmt.Printf("[FRAGMENTED] Paquete %d: %d bytes enviados\n", pktNum, n)
		remaining = remaining[payloadSize:]
		pktNum++
	}
}
