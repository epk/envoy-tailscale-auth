package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	grpcreflect "github.com/bufbuild/connect-grpcreflect-go"
	otelconnect "github.com/bufbuild/connect-opentelemetry-go"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"

	envoy_auth_v3_connect "buf.build/gen/go/envoyproxy/envoy/bufbuild/connect-go/envoy/service/auth/v3/authv3connect"

	"github.com/epk/envoy-tailscale-auth/pkg/authservice"
)

type serverConfig struct {
	// Addr is the address to listen on
	Addr string

	// EnableReflection enables gRPC reflection support
	EnableReflection bool

	// ShutdownTimeout is the timeout to wait for the server to shutdown
	ShutdownTimeout time.Duration

	TLSConfig struct {
		// CertFile is the path to the TLS certificate file
		CertFile string

		// KeyFile is the path to the TLS key file
		KeyFile string
	}
}

// Option is a server configuration option
type Option func(*serverConfig)

// WithAddr sets the address to listen on
func WithAddr(addr string) Option {
	return func(c *serverConfig) {
		c.Addr = addr
	}
}

// WithReflection enables gRPC reflection support
func WithReflection() Option {
	return func(c *serverConfig) {
		c.EnableReflection = true
	}
}

// WithShutdownTimeout sets the timeout to wait for the server to shutdown
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *serverConfig) {
		c.ShutdownTimeout = timeout
	}
}

// WithTLS sets the TLS configuration
func WithTLS(certFile, keyFile string) Option {
	return func(c *serverConfig) {
		c.TLSConfig.CertFile = certFile
		c.TLSConfig.KeyFile = keyFile
	}
}

// New creates a new connect server
func New(opts ...Option) *Server {
	c := &serverConfig{}

	for _, opt := range opts {
		opt(c)
	}
	return &Server{
		config: c,
	}
}

// Server is a connect server
type Server struct {
	config *serverConfig
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	mux := http.NewServeMux()

	// gRPC health check
	checker := grpchealth.NewStaticChecker(
		envoy_auth_v3_connect.AuthorizationName,
	)
	mux.Handle(grpchealth.NewHandler(checker))

	// gRPC reflection support if enabled
	if s.config.EnableReflection {
		reflector := grpcreflect.NewStaticReflector(envoy_auth_v3_connect.AuthorizationName)
		mux.Handle(grpcreflect.NewHandlerV1(reflector))
		mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
	}

	path, handler := envoy_auth_v3_connect.NewAuthorizationHandler(
		authservice.New(),
		otelconnect.WithTelemetry(),
	)
	mux.Handle(path, handler)

	srv := &http.Server{
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,

		Addr:    s.config.Addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	g.Go(func() error {
		if s.config.TLSConfig.CertFile != "" && s.config.TLSConfig.KeyFile != "" {
			if err := srv.ListenAndServeTLS(s.config.TLSConfig.CertFile, s.config.TLSConfig.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}

		return nil
	})

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	return g.Wait()
}
