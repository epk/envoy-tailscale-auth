package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/pflag"

	"github.com/epk/envoy-tailscale-auth/pkg/server"
)

var (
	listenAddr       = pflag.String("addr", ":50051", "address to listen on")
	tlsCertFile      = pflag.String("tls-cert-file", "", "path to the TLS certificate file")
	tlsKeyFile       = pflag.String("tls-key-file", "", "path to the TLS key file")
	enableReflection = pflag.Bool("enable-reflection", true, "enable gRPC reflection support")
	shutdownTimeout  = pflag.Duration("shutdown-timeout", 5*time.Second, "timeout to wait for the server to shutdown")
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	pflag.Parse()

	opts := []server.Option{
		server.WithAddr(*listenAddr),
		server.WithShutdownTimeout(*shutdownTimeout),
		server.WithTLS(*tlsCertFile, *tlsKeyFile),
	}

	if *enableReflection {
		opts = append(opts, server.WithReflection())
	}

	err := server.New(opts...).ListenAndServe(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
