package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	grpc "google.golang.org/grpc"

	"github.com/figment-networks/indexer-manager/worker/connectivity"
	grpcIndexer "github.com/figment-networks/indexer-manager/worker/transport/grpc"
	grpcProtoIndexer "github.com/figment-networks/indexer-manager/worker/transport/grpc/indexer"
	"github.com/figment-networks/indexing-engine/health"
	"github.com/figment-networks/indexing-engine/metrics"
	"github.com/figment-networks/indexing-engine/metrics/prometheusmetrics"
	httpStore "github.com/figment-networks/indexing-engine/worker/store/transport/http"
	"github.com/figment-networks/terra-worker/api"
	"github.com/figment-networks/terra-worker/client"
	"github.com/figment-networks/terra-worker/cmd/common/logger"
	"github.com/figment-networks/terra-worker/cmd/terra-worker/config"
)

type flags struct {
	configPath  string
	showVersion bool
}

var configFlags = flags{}

func init() {
	flag.BoolVar(&configFlags.showVersion, "v", false, "Show application version")
	flag.StringVar(&configFlags.configPath, "config", "", "Path to config")
	flag.Parse()
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	// Initialize configuration
	cfg, err := initConfig(configFlags.configPath)
	if err != nil {
		log.Fatalf("error initializing config [ERR: %v]", err.Error())
	}

	if cfg.RollbarServerRoot == "" {
		cfg.RollbarServerRoot = "github.com/figment-networks/terra-worker"
	}
	rcfg := &logger.RollbarConfig{
		AppEnv:             cfg.AppEnv,
		RollbarAccessToken: cfg.RollbarAccessToken,
		RollbarServerRoot:  cfg.RollbarServerRoot,
		Version:            config.GitSHA,
		ChainIDs:           []string{cfg.ChainID},
	}

	if cfg.AppEnv == "development" || cfg.AppEnv == "local" {
		logger.Init("console", "debug", []string{"stderr"}, rcfg)
	} else {
		logger.Init("json", "info", []string{"stderr"}, rcfg)
	}

	logger.Info(config.IdentityString())
	defer logger.Sync()

	// Initialize metrics
	prom := prometheusmetrics.New()
	err = metrics.AddEngine(prom)
	if err != nil {
		logger.Error(err)
	}
	err = metrics.Hotload(prom.Name())
	if err != nil {
		logger.Error(err)
	}

	workerRunID, err := uuid.NewRandom() // UUID V4
	if err != nil {
		logger.Error(fmt.Errorf("error generating UUID: %w", err))
		return
	}

	managers := strings.Split(cfg.Managers, ",")
	hostname := cfg.Hostname
	if hostname == "" {
		hostname = cfg.Address
	}

	logger.Info(fmt.Sprintf("Self-hostname (%s) is %s:%s ", workerRunID.String(), hostname, cfg.Port))

	c := connectivity.NewWorkerConnections(workerRunID.String(), hostname+":"+cfg.Port, "terra", cfg.ChainID, "0.0.1")
	for _, m := range managers {
		c.AddManager(m + "/client_ping")
	}

	logger.Info(fmt.Sprintf("Connecting to managers (%s)", strings.Join(managers, ",")))

	go c.Run(ctx, logger.GetLogger(), cfg.ManagerInterval)

	grpcServer := grpc.NewServer()

	rpcClient := api.NewClient(cfg.TerraRPCAddr, cfg.DatahubKey, cfg.ChainID, logger.GetLogger(), nil, int(cfg.RequestsPerSecond))
	lcdClient := api.NewClient(cfg.TerraLCDAddr, cfg.DatahubKey, cfg.ChainID, logger.GetLogger(), nil, int(cfg.RequestsPerSecond))

	storeEndpoints := strings.Split(cfg.StoreHTTPEndpoints, ",")
	hStore := httpStore.NewHTTPStore(storeEndpoints, &http.Client{})
	workerClient := client.NewIndexerClient(ctx, logger.GetLogger(), lcdClient, rpcClient, hStore, uint64(cfg.MaximumHeightsToGet))

	worker := grpcIndexer.NewIndexerServer(ctx, workerClient, logger.GetLogger())
	grpcProtoIndexer.RegisterIndexerServiceServer(grpcServer, worker)

	mux := http.NewServeMux()
	attachProfiling(mux)
	attachDynamic(ctx, mux)

	monitor := &health.Monitor{}
	go monitor.RunChecks(ctx, cfg.HealthCheckInterval)
	monitor.AttachHttp(mux)

	mux.Handle("/metrics", metrics.Handler())

	s := &http.Server{
		Addr:         "0.0.0.0:" + cfg.HTTPPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	osSig := make(chan os.Signal)
	exit := make(chan string, 2)
	signal.Notify(osSig, syscall.SIGTERM)
	signal.Notify(osSig, syscall.SIGINT)

	go runGRPC(grpcServer, cfg.Port, logger.GetLogger(), exit)
	go runHTTP(s, cfg.HTTPPort, logger.GetLogger(), exit)

RunLoop:
	for {
		select {
		case sig := <-osSig:
			logger.Info("Stopping worker... ", zap.String("signal", sig.String()))
			cancel()
			logger.Info("Canceled context, gracefully stopping grpc")
			grpcServer.Stop()
			logger.Info("Stopped grpc, stopping http")
			err := s.Shutdown(ctx)
			if err != nil {
				logger.GetLogger().Error("Error stopping http server ", zap.Error(err))
			}
			break RunLoop
		case k := <-exit:
			logger.Info("Stopping worker... ", zap.String("reason", k))
			cancel()
			if k == "grpc" { // (lukanus): when grpc is finished, stop http and vice versa
				err := s.Shutdown(ctx)
				if err != nil {
					logger.GetLogger().Error("Error stopping http server ", zap.Error(err))
				}
			} else {
				grpcServer.Stop()
			}
			break RunLoop
		}
	}

}

func initConfig(path string) (*config.Config, error) {
	cfg := &config.Config{}
	if path != "" {
		if err := config.FromFile(path, cfg); err != nil {
			return nil, err
		}
	}

	if cfg.TerraRPCAddr != "" && (cfg.ChainID == "columbus-3" || cfg.ChainID == "columbus-4") {
		return cfg, nil
	}

	if err := config.FromEnv(cfg); err != nil {
		return nil, err
	}

	if cfg.ChainID != "columbus-3" && cfg.ChainID != "columbus-4" {
		return nil, fmt.Errorf("ChainID must be one of: columbus-3, columbus-4")
	}

	return cfg, nil
}

func runGRPC(grpcServer *grpc.Server, port string, logger *zap.Logger, exit chan<- string) {
	defer logger.Sync()

	logger.Info(fmt.Sprintf("[GRPC] Listening on 0.0.0.0:%s", port))
	lis, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		logger.Error("[GRPC] failed to listen", zap.Error(err))
		exit <- "grpc"
		return
	}

	// (lukanus): blocking call on grpc server
	grpcServer.Serve(lis)
	exit <- "grpc"
}

func runHTTP(s *http.Server, port string, logger *zap.Logger, exit chan<- string) {
	defer logger.Sync()

	logger.Info(fmt.Sprintf("[HTTP] Listening on 0.0.0.0:%s", port))
	if err := s.ListenAndServe(); err != nil {
		logger.Error("[HTTP] failed to listen", zap.Error(err))
	}
	exit <- "http"
}
