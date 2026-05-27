package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/spf13/viper"
    "github.com/trusttrace/trusttrace/internal/consensus"
    "github.com/trusttrace/trusttrace/internal/storage"
    "go.uber.org/zap"
)

func main() {
    log := mustLogger()
    defer log.Sync() //nolint:errcheck

    viper.SetConfigName("consensus")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./config")
    viper.AddConfigPath("/etc/trusttrace")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        log.Fatal("config load failed", zap.Error(err))
    }

    // Storage backend.
    clickhouseDSN := viper.GetString("clickhouse_dsn")
    chClient, err := storage.NewClickHouseClient(clickhouseDSN)
    if err != nil {
        log.Fatal("clickhouse init failed", zap.Error(err))
    }

    // Consensus pipeline.
    quorum, verifiedCh := consensus.NewQuorumManager(2 * time.Minute)
    dedup := consensus.NewDeduplicator()
    storeWorker := consensus.NewStoreWorker(verifiedCh, chClient, log)

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // Start store worker in background.
    go storeWorker.Run(ctx)

    // Prometheus / health.
    go func() {
        mux := http.NewServeMux()
        mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
            fmt.Fprintln(w, "ok")
        })
        port := viper.GetString("metrics_port")
        if port == "" {
            port = "9091"
        }
        http.ListenAndServe(":"+port, mux) //nolint:errcheck
    }()

    // gRPC server (blocking).
    grpcAddr := viper.GetString("grpc_addr")
    if grpcAddr == "" {
        grpcAddr = ":50051"
    }
    srv := consensus.NewGRPCServer(quorum, dedup, log)

    go func() {
        <-ctx.Done()
        log.Info("shutdown signal received")
    }()

    log.Info("TrustTrace consensus engine starting", zap.String("grpc_addr", grpcAddr))
    if err := srv.ListenAndServe(grpcAddr); err != nil {
        log.Fatal("gRPC server error", zap.Error(err))
    }
}

func mustLogger() *zap.Logger {
    l, _ := zap.NewProduction()
    return l
}
