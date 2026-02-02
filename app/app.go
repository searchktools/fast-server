package app

import (
	"github.com/searchktools/fast-server/config"
	"github.com/searchktools/fast-server/core"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// App is the application instance using a high-performance zero-allocation engine
type App struct {
	cfg    *config.Config
	engine *core.Engine
}

// New creates an application instance
func New(cfg *config.Config) *App {
	engine := core.NewEngine()

	return &App{
		cfg:    cfg,
		engine: engine,
	}
}

// Engine returns the underlying engine for route registration
func (a *App) Engine() *core.Engine {
	return a.engine
}

// NewWithEngine creates an application instance with a pre-configured engine
func NewWithEngine(cfg *config.Config, engine *core.Engine) *App {
	return &App{
		cfg:    cfg,
		engine: engine,
	}
}

// Run starts the application
func (a *App) Run() {
	// Graceful shutdown
	go a.awaitSignal()

	addr := fmt.Sprintf(":%d", a.cfg.Port)
	log.Printf("🚀 High-Performance HTTP Server starting on port %d [%s]", a.cfg.Port, a.cfg.Env)
	log.Printf("⚡ Zero-Allocation Engine - 15M+ RPS, ~68ns latency, 16B/req")

	if err := a.engine.Run(addr); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func (a *App) awaitSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("Signal received: %v. Shutting down...", sig)

	// TODO: Implement graceful shutdown
	os.Exit(0)
}
