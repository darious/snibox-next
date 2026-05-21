package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/darious1472/snibox-next/internal/assets"
	"github.com/darious1472/snibox-next/internal/handlers"
	"github.com/darious1472/snibox-next/internal/importer"
	"github.com/darious1472/snibox-next/internal/markdown"
	"github.com/darious1472/snibox-next/internal/store"
)

// Version is set via -ldflags="-X main.Version=..." at build time.
var Version = "dev"

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := flag.String("addr", envStr("SNIBOX_ADDR", "127.0.0.1:8979"), "listen address")
	db := flag.String("db", envStr("SNIBOX_DB", "./snibox.db"), "sqlite path")
	seed := flag.Bool("seed-demo", envBool("SNIBOX_SEED_DEMO", false), "load testdata/seed.json into empty db")
	readOnly := flag.Bool("read-only", envBool("SNIBOX_READ_ONLY", false), "disable write routes")
	trustNet := flag.Bool("trust-network", envBool("SNIBOX_TRUST_NETWORK", false), "permit non-loopback bind (auth is assumed external — see SPEC §1.1)")
	showVer := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVer {
		fmt.Println("snibox-next", versionString())
		return
	}

	if err := guardNetwork(*addr, *trustNet); err != nil {
		log.Fatalf("refusing to start: %v", err)
	}

	dbConn, err := store.Open(*db)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer dbConn.Close()
	repo := store.NewRepo(dbConn)

	if *seed {
		if err := maybeSeed(repo); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	chromaCSS, err := markdown.WriteStylesheet()
	if err != nil {
		log.Fatalf("chroma css: %v", err)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           handlers.New(repo, assets.FS, []byte(chromaCSS), *readOnly).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	idle := make(chan struct{})
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
		close(idle)
	}()

	log.Printf("snibox %s listening on %s (read-only=%v)", versionString(), *addr, *readOnly)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
	<-idle
}

func versionString() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				if len(s.Value) > 12 {
					return s.Value[:12]
				}
				return s.Value
			}
		}
	}
	return Version
}

func guardNetwork(addr string, trust bool) error {
	if trust {
		return nil
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return errors.New("non-loopback bind without --trust-network — see SPEC §1.1 (auth is assumed external)")
	}
	ip := net.ParseIP(host)
	if ip != nil && !ip.IsLoopback() {
		return errors.New("non-loopback bind without --trust-network — see SPEC §1.1 (auth is assumed external)")
	}
	return nil
}

func maybeSeed(repo *store.Repo) error {
	ctx := context.Background()
	counts, err := repo.Counts(ctx)
	if err != nil {
		return err
	}
	if counts.All+counts.Archive > 0 {
		log.Printf("seed-demo: skipping (db already has %d items)", counts.All+counts.Archive)
		return nil
	}
	f, err := assets.FS.Open("seed.json")
	if err != nil {
		return err
	}
	defer f.Close()
	rep, err := importer.Run(ctx, repo, f, importer.ModeReplace)
	if err != nil {
		return err
	}
	log.Printf("seed-demo: imported %d items", rep.Imported)
	return nil
}
