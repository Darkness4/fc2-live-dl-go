// Package ws provides a WebSocket proxy server to forward messages between a client and a backend server.
package ws

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/rs/zerolog/log"
)

// Proxy struct to handle WebSocket connections and commands.
type Proxy struct {
	target     string             // Target WebSocket server URL
	hclient    *http.Client       // HTTP client to dial the backend WebSocket server
	clientConn *websocket.Conn    // WebSocket connection to the client
	ctx        context.Context    // Context to manage the connection lifecycle
	cancelFunc context.CancelFunc // Cancel function to stop the proxy gracefully
}

// NewProxy creates a new Proxy instance.
func NewProxy(target string, hclient *http.Client) *Proxy {
	return &Proxy{
		target:  target,
		hclient: hclient,
	}
}

// Start begins the WebSocket proxy server.
func (p *Proxy) Start(w http.ResponseWriter, r *http.Request) {
	// Set up the WebSocket connection with the client
	var err error
	p.ctx, p.cancelFunc = context.WithCancel(context.Background())
	p.clientConn, err = websocket.Accept(w, r, nil)
	if err != nil {
		log.Err(err).Msg("Failed to accept WebSocket connection")
		return
	}
	defer p.clientConn.Close(websocket.StatusInternalError, "Internal Error")

	// Simulate backend WebSocket connection (replace with real backend connection if needed)
	backendConn, _, err := websocket.Dial(p.ctx, p.target, &websocket.DialOptions{
		HTTPClient: p.hclient,
	})
	if err != nil {
		log.Err(err).Msg("Failed to dial backend WebSocket server")
		p.clientConn.Close(websocket.StatusInternalError, "Internal Error")
		return
	}
	backendConn.SetReadLimit(10485760) // 10 MiB
	defer backendConn.Close(websocket.StatusInternalError, "Internal Error")

	// Goroutine to handle forwarding between client and backend
	go p.forwardMessages(p.clientConn, backendConn, "client")

	// Goroutine to handle forwarding between backend and client
	go p.forwardMessages(backendConn, p.clientConn, "backend")

	// Block until context is canceled (proxy stopped)
	<-p.ctx.Done()

	// Cleanly close the WebSocket connection
	p.clientConn.Close(websocket.StatusNormalClosure, "Proxy Stopped")
	log.Info().Msg("Proxy Stopped")
}

// forwardMessages forwards messages between the client and backend.
func (p *Proxy) forwardMessages(srcConn, dstConn *websocket.Conn, source string) {
	defer func() {
		// When one connection closes, cancel the context to stop the proxy
		p.cancelFunc()
	}()

	for {
		// Read message from source (client or backend)
		var message any
		err := wsjson.Read(p.ctx, srcConn, &message)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure ||
				errors.Is(err, net.ErrClosed) {
				log.Err(err).Msg("Failed to read message")
			}
			break
		}
		// Write message to destination (backend or client)
		err = wsjson.Write(p.ctx, dstConn, message)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure ||
				errors.Is(err, net.ErrClosed) {
				log.Err(err).Msg("Failed to write message")
			}
			break
		}
		log.Trace().Any("message", message).Str("source", source).Msg("Message forwarded")
	}
}

// SendMessage sends a JSON message to the client.
func (p *Proxy) SendMessage(msg any) {
	err := wsjson.Write(p.ctx, p.clientConn, msg)
	if err != nil {
		log.Err(err).Msg("Failed to send message")
	}
}

// HTTP handler to start the WebSocket proxy.
func proxyHandler(target string, hclient *http.Client, commandChan chan any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy := NewProxy(target, hclient)

		go func() {
			for {
				select {
				case cmd := <-commandChan:
					proxy.SendMessage(cmd)
				case <-proxy.ctx.Done():
					return
				}
			}
		}()

		// Start the WebSocket proxy
		proxy.Start(w, r)
	}
}

// Server struct to manage the WebSocket proxy server.
type Server struct {
	*httptest.Server
	commandChan chan any
}

// NewServer creates a new WebSocket proxy server.
func NewServer(targetURL string, hclient *http.Client) *Server {
	commandChan := make(chan any)
	return &Server{
		Server:      httptest.NewServer(proxyHandler(targetURL, hclient, commandChan)),
		commandChan: commandChan,
	}
}

// SendMessage sends a JSON message to the WebSocket client.
func (s *Server) SendMessage(msg any) {
	s.commandChan <- msg
}
