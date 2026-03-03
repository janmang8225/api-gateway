package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/janmang8225/api-gateway/internal/balancer"
	"github.com/janmang8225/api-gateway/internal/breaker"
	"github.com/janmang8225/api-gateway/internal/config"
	"github.com/janmang8225/api-gateway/internal/logger"
	"github.com/janmang8225/api-gateway/internal/metrics"
	"github.com/janmang8225/api-gateway/internal/middleware/auth"
	"github.com/janmang8225/api-gateway/internal/middleware/ratelimit"
	"github.com/janmang8225/api-gateway/internal/proxy"
)

type routeHandler struct {
	balancer *balancer.RoundRobin
	breakers map[string]*breaker.CircuitBreaker
	auth     bool
	path     string
}

func buildRoutes(cfg *config.Config) []*routeHandler {
	var handlers []*routeHandler

	for _, route := range cfg.Routes {
		rb := balancer.NewRoundRobin(route.Backends)

		breakers := make(map[string]*breaker.CircuitBreaker)
		for _, backend := range route.Backends {
			breakers[backend] = breaker.New(3, 2, 10*time.Second)
		}

		handlers = append(handlers, &routeHandler{
			balancer: rb,
			breakers: breakers,
			auth:     route.Auth,
			path:     route.Path,
		})
	}

	// sort by path length descending so more specific routes match first
	sort.Slice(handlers, func(i, j int) bool {
		return len(handlers[i].path) > len(handlers[j].path)
	})

	return handlers
}

func findRoute(handlers []*routeHandler, path string) *routeHandler {
	for _, rh := range handlers {
		if path == rh.path || len(path) > len(rh.path) && path[len(rh.path)] == '/' {
			return rh
		}
	}
	return nil
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	printVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *printVersion {
		fmt.Println("api-gateway v1.0.0")
		os.Exit(0)
	}

	cfgManager, err := config.NewManager(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	m := metrics.New()
	p := proxy.New()

	cfg := cfgManager.Get()
	jwtMiddleware := auth.NewJWTMiddleware(cfg.JWTSecret)
	routes := buildRoutes(cfg)

	mux := http.NewServeMux()

	// metrics endpoint
	mux.Handle("/metrics", m.Handler())

	// single catch-all — we do our own prefix matching
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rh := findRoute(routes, r.URL.Path)
		if rh == nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 Not Found"))
			return
		}

		backend := rh.balancer.Next()
		cb := rh.breakers[backend]
		handler := p.Forward(backend, cb)

		if rh.auth {
			jwtMiddleware.Middleware(handler).ServeHTTP(w, r)
		} else {
			handler.ServeHTTP(w, r)
		}
	})

	rl := ratelimit.NewRateLimiter(10, 5)

	addr := fmt.Sprintf(":%d", cfg.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      logger.Middleware(rl.Middleware(mux), m),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		fmt.Printf("api-gateway v1.0.0\n")
		fmt.Printf("config:  %s\n", *configPath)
		fmt.Printf("listening on %s\n", addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	for {
		sig := <-quit

		if sig == syscall.SIGHUP {
			fmt.Println("received SIGHUP — reloading config...")
			err := cfgManager.Reload()
			if err != nil {
				log.Printf("config reload error: %v", err)
				continue
			}
			newCfg := cfgManager.Get()
			routes = buildRoutes(newCfg)
			jwtMiddleware = auth.NewJWTMiddleware(newCfg.JWTSecret)
			fmt.Println("config reloaded.")
			continue
		}

		break
	}

	fmt.Println("\nshutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	fmt.Println("stopped cleanly.")
}