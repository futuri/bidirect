package websocket

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/example/bidirect/internal/logging"
	"golang.org/x/net/websocket"
)

type Server struct {
	port       int
	ringBuffer *RingBuffer
	httpServer *http.Server
	wg         sync.WaitGroup
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
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.serveHTML)
	mux.HandleFunc("/client.js", s.serveJS)
	mux.Handle("/stream", websocket.Handler(s.handleWebSocket))

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		logging.Infof("WebSocket server listening on :%d", s.port)
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			logging.Errorf("HTTP server error: %v", err)
		}
	}()

	return nil
}

func (s *Server) Stop() {
	close(s.stopCh)
	if s.httpServer != nil {
		s.httpServer.Close()
	}
	s.wg.Wait()
}

func (s *Server) GetRingBuffer() *RingBuffer {
	return s.ringBuffer
}

func (s *Server) handleWebSocket(ws *websocket.Conn) {
	defer ws.Close()
	logging.Infof("WebSocket client connected: %s", ws.Request().RemoteAddr)

	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		var sizeBytes [4]byte
		if _, err := io.ReadFull(ws, sizeBytes[:]); err != nil {
			if err != io.EOF {
				logging.Errorf("Error reading size: %v", err)
			}
			return
		}

		size := binary.LittleEndian.Uint32(sizeBytes[:])
		if size == 0 || size > 10*1024*1024 {
			logging.Errorf("Invalid frame size: %d", size)
			return
		}

		webpData := make([]byte, size)
		if _, err := io.ReadFull(ws, webpData); err != nil {
			logging.Errorf("Error reading frame data: %v", err)
			return
		}

		if err := s.processFrame(webpData); err != nil {
			logging.Errorf("Error processing frame: %v", err)
			continue
		}
	}
}

func (s *Server) processFrame(webpData []byte) error {
	bgraData, width, height, err := DecodeImageToBGRA(webpData)
	if err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	s.ringBuffer.Write(bgraData, width, height)
	return nil
}

func (s *Server) serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlPage))
}

func (s *Server) serveJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write([]byte(jsClient))
}
