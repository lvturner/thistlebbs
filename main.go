// Command thistlebbs runs the Thistle BBS telnet server.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"thistlebbs/internal/server"
	"thistlebbs/internal/store"
)

func main() {
	var (
		listen     = flag.String("listen", ":1997", "address to listen on (host:port)")
		dbPath     = flag.String("db", "data/thistlebbs.db", "path to the SQLite database file")
		board      = flag.String("name", "Thistle BBS", "display name of the system")
		bannerPath = flag.String("banner", "data/banner.txt", "path to banner file (empty = built-in)")
		menuDir    = flag.String("menu-dir", "data/menus", "path to menu templates directory (empty = no templates)")
		version    = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *version {
		log.Printf("thistlebbs 0.1.0")
		return
	}

	flag.Usage = func() {
		log.Printf("thistlebbs - a telnet BBS server")
		flag.PrintDefaults()
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer st.Close()

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("Thistle BBS listening on %s (board: %q, db: %s)",
		ln.Addr(), *board, *dbPath)

	cfg := server.Config{BoardName: *board, Store: st, BannerPath: *bannerPath, MenuDir: *menuDir}
	srv := server.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		errc <- srv.Serve(ln)
	}()

	select {
	case err := <-errc:
		if err != nil {
			log.Fatalf("server: %v", err)
		}
	case <-ctx.Done():
		log.Printf("shutting down...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
		log.Printf("bye.")
	}
}