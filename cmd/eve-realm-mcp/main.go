package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	mcpv1 "github.com/beduardo/eve-realm-mcp/gen/proto/mcp/v1"
	internalmcp "github.com/beduardo/eve-realm-mcp/internal/mcp"
	"github.com/beduardo/eve-realm-mcp/internal/logging"
	"github.com/beduardo/eve-realm-mcp/internal/registry"
	"github.com/beduardo/eve-realm-mcp/internal/tools"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// Version, GitHash, and BuildDate are populated via -ldflags at build time.
// They default to these values when the binary is built without ldflags injection.
var (
	Version   = "dev"
	GitHash   = "unknown"
	BuildDate = "unknown"
)

// StartupMessage returns the canonical startup log line for this binary.
// Format: eve-realm-mcp online (v<Version>, <GitHash>, <BuildDate>)
func StartupMessage() string {
	return fmt.Sprintf("eve-realm-mcp online (v%s, %s, %s)", Version, GitHash, BuildDate)
}

// versionResponse is the JSON schema for the /version endpoint.
type versionResponse struct {
	Version   string `json:"version"`
	GitHash   string `json:"git_hash"`
	BuildDate string `json:"build_date"`
}

// VersionHandler returns an http.Handler that serves GET /version with a JSON
// body containing the current version, git hash, and build date.
func VersionHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(versionResponse{
			Version:   Version,
			GitHash:   GitHash,
			BuildDate: BuildDate,
		})
	})
}

// healthResponse is the JSON schema for the /healthz and /readyz endpoints.
type healthResponse struct {
	Status string `json:"status"`
}

// HealthzHandler returns an http.Handler that serves GET /healthz with a JSON
// body of {"status":"ok"} and HTTP 200, indicating the process is alive.
func HealthzHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	})
}

// ReadyzHandler returns an http.Handler that serves GET /readyz with a JSON
// body of {"status":"ok"} and HTTP 200, indicating the server is ready to
// accept traffic.
func ReadyzHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	})
}

// ShutdownServer logs the canonical shutdown message and then calls
// srv.Shutdown(ctx) to drain active connections gracefully.
func ShutdownServer(ctx context.Context, srv *http.Server, logger *slog.Logger) {
	logger.Info("eve-realm-mcp shutting down")
	srv.Shutdown(ctx) //nolint:errcheck
}

// pingTool returns the registry.Tool descriptor for the built-in ping diagnostic
// tool. It delegates to the internal tools package.
func pingTool() registry.Tool {
	return tools.NewTool()
}

// startServers constructs the ToolRegistry, registers the ping tool, creates the
// HTTP and gRPC servers, and starts both under a shared errgroup. When both
// listeners are bound and ready to accept connections, ready is closed. The
// function blocks until the context is cancelled, at which point both servers are
// shut down gracefully before returning.
//
// ready may be nil; if non-nil it is closed exactly once when both servers are
// ready.
func startServers(ctx context.Context, httpPort, grpcPort int, logger *slog.Logger, ready chan<- struct{}) error {
	// Build the tool registry and register built-in tools before any server
	// starts, so ListTools returns correct results immediately after readiness.
	reg := registry.NewMapRegistry()
	reg.Register(pingTool())

	// Construct the gRPC server and register the MCPService.
	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(logging.NewInterceptor(logger)))
	mcpv1.RegisterMCPServiceServer(grpcSrv, internalmcp.NewMCPService(reg))

	// Construct the HTTP server.
	mux := http.NewServeMux()
	mux.Handle("/version", VersionHandler())
	mux.Handle("/healthz", HealthzHandler())
	mux.Handle("/readyz", ReadyzHandler())

	httpAddr := fmt.Sprintf(":%d", httpPort)
	httpSrv := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	grpcAddr := fmt.Sprintf(":%d", grpcPort)

	// Pre-bind both listeners before signalling readiness so that callers can
	// connect immediately after the ready channel is closed.
	httpLis, err := net.Listen("tcp", httpAddr)
	if err != nil {
		return fmt.Errorf("http listen %s: %w", httpAddr, err)
	}

	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		httpLis.Close()
		return fmt.Errorf("grpc listen %s: %w", grpcAddr, err)
	}

	logger.Info("listening", "addr", httpAddr)
	logger.Info("grpc listening", "addr", grpcAddr)

	// Signal readiness now that both listeners are bound.
	if ready != nil {
		close(ready)
	}

	eg, egCtx := errgroup.WithContext(ctx)

	// HTTP server goroutine.
	eg.Go(func() error {
		if err := httpSrv.Serve(httpLis); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	// gRPC server goroutine.
	eg.Go(func() error {
		if err := grpcSrv.Serve(grpcLis); err != nil {
			return fmt.Errorf("grpc server: %w", err)
		}
		return nil
	})

	// Shutdown goroutine: wait for context cancellation then stop both servers.
	eg.Go(func() error {
		<-egCtx.Done()
		logger.Info("eve-realm-mcp shutting down")
		httpSrv.Shutdown(context.Background()) //nolint:errcheck
		grpcSrv.GracefulStop()
		return nil
	})

	return eg.Wait()
}

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	grpcPort := flag.Int("grpc-port", 50051, "gRPC server port")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info(StartupMessage())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := startServers(ctx, *port, *grpcPort, logger, nil); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
