package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/FsocietyVoid/TrustTrace/internal/notary"
	"github.com/FsocietyVoid/TrustTrace/internal/storage"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	log := mustLogger()
	defer log.Sync() //nolint:errcheck

	viper.SetConfigName("notary")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/trusttrace")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("config load failed", zap.Error(err))
	}

	// ClickHouse storage.
	chClient, err := storage.NewClickHouseClient(viper.GetString("clickhouse_dsn"))
	if err != nil {
		log.Fatal("clickhouse init", zap.Error(err))
	}

	// Ethereum anchor.
	ethAnchor, err := notary.NewEthAnchor(
		viper.GetString("eth_rpc_url"),
		viper.GetString("eth_private_key"),
		viper.GetString("contract_address"),
		viper.GetInt64("chain_id"),
		log,
	)
	if err != nil {
		log.Fatal("eth anchor init", zap.Error(err))
	}

	// IPFS store.
	ipfsStore := notary.NewIPFSStore(viper.GetString("ipfs_api"))

	batcher := notary.NewBatcher(chClient, ethAnchor, ipfsStore, log)

	// Health endpoint.
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintln(w, "ok")
		})
		port := viper.GetString("metrics_port")
		if port == "" {
			port = "9092"
		}
		http.ListenAndServe(":"+port, mux) //nolint:errcheck
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("TrustTrace cryptographic notary starting")
	batcher.Run(ctx)
	log.Info("notary stopped")
}

func mustLogger() *zap.Logger {
	l, _ := zap.NewProduction()
	return l
}
