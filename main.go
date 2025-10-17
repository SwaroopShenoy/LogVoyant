package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"logvoyant/internal/ingest"
	"logvoyant/internal/server"
	"logvoyant/internal/storage"
)

//go:embed internal/web/ui
var staticFiles embed.FS

var (
	port     = flag.Int("port", 3100, "HTTP server port")
	groqKey  = flag.String("groq-key", "", "Groq API key for LLM analysis (optional)")
	dbPath   = flag.String("db", "./logvoyant.db", "BoltDB database path")
	discover = flag.Bool("discover", true, "Auto-discover log sources")
)

func main() {
	flag.Parse()

	fmt.Printf(`
╦  ╔═╗╔═╗╦  ╦╔═╗╦ ╦╔═╗╔╗╔╔╦╗
║  ║ ║║ ╦╚╗╔╝║ ║╚╦╝╠═╣║║║ ║ 
╩═╝╚═╝╚═╝ ╚╝ ╚═╝ ╩ ╩ ╩╝╚╝ ╩ 
Context-Aware Log Analysis
`)

	// Initialize storage
	store, err := storage.NewBoltStorage(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize server
	srv := server.New(&server.Config{
		Port:        *port,
		Storage:     store,
		StaticFiles: staticFiles,
		GroqAPIKey:  *groqKey,
	})

	// Start server
	go func() {
		fmt.Printf("\n🚀 LogVoyant running on http://localhost:%d\n\n", *port)
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Auto-discover sources
	if *discover {
		fmt.Println("🔍 Auto-discovering log sources...")
		go func() {
			if err := ingest.DiscoverAndStart(store, srv.Hub()); err != nil {
				log.Printf("Discovery error: %v", err)
			}
		}()
	}

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("\n👋 Shutting down gracefully...")
	srv.Stop()
	fmt.Println("✓ Goodbye!")
}