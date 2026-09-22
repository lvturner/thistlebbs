package server

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"thistlebbs/internal/ansi"
	"thistlebbs/internal/store"
	"thistlebbs/internal/telnet"
)

// Config configures the BBS server.
type Config struct {
	BoardName  string
	Store      *store.Store
	BannerPath string // path to banner.txt; empty = use built-in
	MenuDir    string // path to menus/ directory; empty = no templates

	banner string                 // rendered banner, loaded once in New
	menus  map[string]*MenuTemplate // loaded menu templates, keyed by name
}

// Server accepts telnet connections and runs one session per client.
type Server struct {
	cfg    Config
	ln     net.Listener
	mu     sync.Mutex
	conns  map[net.Conn]struct{}
	wg     sync.WaitGroup
	closed atomic.Bool
	nextID atomic.Int64
}

// New creates a Server. The store must already be opened.
func New(cfg Config) *Server {
	if cfg.BannerPath != "" {
		data, err := os.ReadFile(cfg.BannerPath)
		if err != nil {
			log.Printf("banner: %v (using built-in)", err)
		} else {
			cfg.banner = ansi.ExpandTags(string(data))
		}
	}

	// Load menu templates
	cfg.menus = make(map[string]*MenuTemplate)
	if cfg.MenuDir != "" {
		templates := []string{"gate", "main", "profile", "edit_profile", "reply", "new_thread", "view_thread", "login", "register", "board", "users", "view_user"}
		for _, name := range templates {
			path := filepath.Join(cfg.MenuDir, name+".txt")
			t, err := LoadMenuTemplate(path)
			if err != nil {
				log.Printf("menu %s: %v", name, err)
			} else {
				cfg.menus[name] = t
			}
		}
		if len(cfg.menus) == 0 {
			log.Printf("no menu templates loaded from %s (falling back to code)", cfg.MenuDir)
		} else {
			log.Printf("loaded %d menu templates from %s", len(cfg.menus), cfg.MenuDir)
		}
	}

	return &Server{
		cfg:   cfg,
		conns: make(map[net.Conn]struct{}),
	}
}

// Addr reports the bound listener address (valid after Serve is running).
func (s *Server) Addr() net.Addr {
	if s.ln == nil {
		return nil
	}
	return s.ln.Addr()
}

// Serve accepts connections until the listener fails or Shutdown is called.
func (s *Server) Serve(ln net.Listener) error {
	s.ln = ln
	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.closed.Load() || errors.Is(err, net.ErrClosed) {
				return nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return err
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handle(conn)
		}()
	}
}

// Shutdown stops accepting, closes live connections and waits for handlers.
func (s *Server) Shutdown(ctx context.Context) error {
	s.closed.Store(true)
	if s.ln != nil {
		_ = s.ln.Close()
	}
	s.mu.Lock()
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(30 * time.Second)
	}

	s.mu.Lock()
	s.conns[conn] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()

	id := s.nextID.Add(1)
	log.Printf("conn #%d from %s", id, conn.RemoteAddr())

	sess := newSession(conn, &s.cfg, id)

	// Negotiation: WILL ECHO (server controls all echoing), WILL SGA, then
	// request NAWS / BINARY / TTYPE from the client.
	sess.writeRaw(telnet.Concat(
		telnet.Cmd(telnet.WILL, telnet.OptEcho),
		telnet.Cmd(telnet.WILL, telnet.OptSGA),
		telnet.Cmd(telnet.DO, telnet.OptNAWS),
		telnet.Cmd(telnet.DO, telnet.OptBinary),
		telnet.Cmd(telnet.DO, telnet.OptTTYPE),
		telnet.Subneg(telnet.OptTTYPE, []byte{telnet.TTYPESEND}),
	))

	go sess.readLoop()

	err := sess.run()
	// Empty err means the user hung up deliberately; readLoop reports wire EOF.
	switch {
	case err == nil:
		log.Printf("conn #%d disconnected", id)
	default:
		log.Printf("conn #%d closed: %v", id, err)
	}

	sess.close()
}