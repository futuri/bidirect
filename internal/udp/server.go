package udp

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/example/bidirect/internal/logging"
)

type Server struct {
	port       int
	conn       *net.UDPConn
	ringBuffer *RingBuffer
	stopCh     chan struct{}
}

func NewServer(port int) *Server {
	return &Server{
		port:       port,
		ringBuffer: NewRingBuffer(),
		stopCh:     make(chan struct{}),
	}
}

func (s *Server) Start() error {
	addr := &net.UDPAddr{
		Port: s.port,
		IP:   net.IPv4(0, 0, 0, 0),
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("listen UDP: %w", err)
	}

	s.conn = conn
	conn.SetReadBuffer(8 * 1024 * 1024) // 8MB buffer

	logging.Infof("UDP server listening on 0.0.0.0:%d", s.port)

	go s.receiveLoop()

	return nil
}

func (s *Server) receiveLoop() {
	// Buffer para paquetes fragmentados - aumentado a 10MB
	frameBuffer := make([]byte, 0, 10*1024*1024)
	packetBuf := make([]byte, 65535) // Max UDP packet

	var expectedSize uint32
	var currentSize uint32
	var frameID int64

	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		n, addr, err := s.conn.ReadFromUDP(packetBuf)
		if err != nil {
			continue
		}

		if n < 4 {
			continue
		}

		logging.Infof("UDP packet received from %s: %d bytes", addr.String(), n)

		// Protocolo:
		// Byte 0: tipo (0=inicio frame, 1=continuación, 2=frame completo pequeño)
		// Bytes 1-4: tamaño total (solo en tipo 0)
		// Bytes siguientes: datos

		packetType := packetBuf[0]

		switch packetType {
		case 0: // Inicio de frame
			if n < 5 {
				continue
			}
			expectedSize = binary.LittleEndian.Uint32(packetBuf[1:5])
			frameID++
			frameBuffer = frameBuffer[:0]
			frameBuffer = append(frameBuffer, packetBuf[5:n]...)
			currentSize = uint32(n - 5)
			logging.Infof("[%d] Frame start: %d bytes expected, %d received", frameID, expectedSize, currentSize)

		case 1: // Continuación
			if expectedSize == 0 {
				logging.Infof("Frame continuation sin inicio")
				continue
			}
			frameBuffer = append(frameBuffer, packetBuf[1:n]...)
			currentSize += uint32(n - 1)
			logging.Infof("[%d] Frame continue: %d/%d bytes (pkt: %d)", frameID, currentSize, expectedSize, n)

		case 2: // Frame completo pequeño (cabe en un paquete)
			if n < 5 {
				continue
			}
			size := binary.LittleEndian.Uint32(packetBuf[1:5])
			frameID++
			logging.Infof("[%d] Frame complete: %d bytes", frameID, size)
			if uint32(n-5) >= size {
				s.processFrame(packetBuf[5 : 5+size])
			}
			expectedSize = 0
			currentSize = 0
			continue
		}

		// Verificar si frame completo
		if expectedSize > 0 && currentSize >= expectedSize {
			if currentSize > expectedSize {
				logging.Infof("[%d] WARNING: recibió %d bytes, esperaba %d (descartando %d)", frameID, currentSize, expectedSize, currentSize-expectedSize)
			}
			logging.Infof("[%d] Frame assembled: %d bytes", frameID, expectedSize)
			s.processFrame(frameBuffer[:expectedSize])
			expectedSize = 0
			currentSize = 0
		}
	}
}

func (s *Server) processFrame(data []byte) {
	logging.Infof("Processing frame: %d bytes", len(data))
	bgra, width, height, err := DecodeImageToBGRA(data)
	if err != nil {
		logging.Errorf("Decode frame error: %v", err)
		return
	}

	logging.Infof("Frame decoded: %dx%d", width, height)
	s.ringBuffer.Write(bgra, width, height)
}

func (s *Server) GetRingBuffer() *RingBuffer {
	return s.ringBuffer
}

func (s *Server) Stop() {
	close(s.stopCh)
	if s.conn != nil {
		s.conn.Close()
	}
}
