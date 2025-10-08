package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Proxy represents the database proxy server
type Proxy struct {
	config   Config
	listener net.Listener
	sessions map[string]*Session
	mu       sync.RWMutex
	stopChan chan struct{}
}

// Session represents a client session
type Session struct {
	ID           string
	PseudoUser   string
	RealCreds    Credentials
	ClientConn   net.Conn
	DBConn       net.Conn
	StartTime    time.Time
	QueryCount   int
}

// NewProxy creates a new proxy instance
func NewProxy(config Config) *Proxy {
	return &Proxy{
		config:   config,
		sessions: make(map[string]*Session),
		stopChan: make(chan struct{}),
	}
}

// Start starts the proxy server
func (p *Proxy) Start() error {
	addr := fmt.Sprintf("%s:%s", p.config.ProxyHost, p.config.ProxyPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	p.listener = listener

	for {
		select {
		case <-p.stopChan:
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				// Check if we're shutting down
				select {
				case <-p.stopChan:
					return nil
				default:
					fmt.Printf("Error accepting connection: %v\n", err)
					continue
				}
			}
			
			// Handle connection in goroutine
			go p.handleConnection(conn)
		}
	}
}

// Stop stops the proxy server
func (p *Proxy) Stop() {
	close(p.stopChan)
	if p.listener != nil {
		p.listener.Close()
	}
	
	// Close all active sessions
	p.mu.Lock()
	for _, session := range p.sessions {
		if session.ClientConn != nil {
			session.ClientConn.Close()
		}
		if session.DBConn != nil {
			session.DBConn.Close()
		}
	}
	p.mu.Unlock()
}

// handleConnection handles a new client connection
func (p *Proxy) handleConnection(clientConn net.Conn) {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	
	color.New(color.FgHiCyan).Printf("\n[%s] 🔗 New connection from %s\n", sessionID, clientConn.RemoteAddr())
	
	defer func() {
		clientConn.Close()
		p.removeSession(sessionID)
		color.New(color.FgHiBlack).Printf("[%s] 🔌 Connection closed\n", sessionID)
	}()

	// Handle PostgreSQL protocol
	handler := NewPostgreSQLHandler(p.config, sessionID)
	if err := handler.Handle(clientConn); err != nil {
		if err != io.EOF {
			color.Red("[%s] ❌ Error: %v\n", sessionID, err)
		}
	}
}

func (p *Proxy) removeSession(sessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.sessions, sessionID)
}
