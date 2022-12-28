package server

import "time"

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
