package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FsocietyVoid/TrustTrace/internal/prober"
	"github.com/FsocietyVoid/TrustTrace/pkg/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	log := mustLogger()
	defer log.Sync() //nolint:errcheck

	viper.SetConfigName("prober")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/trusttrace")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("config load failed", zap.Error(err))
	}

	cfg := prober.DefaultConfig()
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal("config parse failed", zap.Error(err))
	}

	if cfg.Region == "" {
		log.Fatal("region must be set (e.g. us-east-1)")
	}

	// Prometheus metrics server.
	reg := prometheus.NewRegistry()
	telemetry.NewMetrics(reg)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", telemetry.Handler())
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintln(w, "ok")
		})
		port := viper.GetString("metrics_port")
		if port == "" {
			port = "9090"
		}
		log.Info("metrics server", zap.String("port", port))
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Error("metrics server error", zap.Error(err))
		}
	}()

	pool, err := prober.NewPool(cfg, log)
	if err != nil {
		log.Fatal("pool init failed", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("TrustTrace edge prober starting",
		zap.String("region", cfg.Region),
		zap.Int("targets", len(cfg.Targets)),
		zap.Int("workers", cfg.WorkerCount),
	)
	pool.Run(ctx)
	log.Info("prober stopped gracefully")

	// Drain briefly.
	time.Sleep(500 * time.Millisecond)
}

func mustLogger() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	l, err := cfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger init: %v\n", err)
		os.Exit(1)
	}
	return l
}
