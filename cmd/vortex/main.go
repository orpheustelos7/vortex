package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/orpheustelos7/vortex/internal/config"
	"github.com/orpheustelos7/vortex/internal/gateway"
	"github.com/orpheustelos7/vortex/internal/observability"
	"github.com/orpheustelos7/vortex/internal/ratelimiter"
	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	listenAddr := getenv("LISTEN_ADDR", ":8080")
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	etcdEndpoints := strings.Split(getenv("ETCD_ENDPOINTS", "localhost:2379"), ",")

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis unavailable: %v", err)
	}

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to connect to etcd: %v", err)
	}
	defer etcdClient.Close()

	store := config.NewStore()
	watcher := config.NewWatcher(etcdClient, store)
	if err := watcher.LoadInitial(context.Background()); err != nil {
		log.Fatalf("failed loading policies: %v", err)
	}
	watchCtx, cancelWatch := context.WithCancel(context.Background())
	defer cancelWatch()
	go watcher.Watch(watchCtx)

	metrics := observability.NewMetrics()
	limiter := ratelimiter.NewRedisTokenBucket(redisClient)
	server := gateway.New(store, limiter, metrics)
	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: server.Handler(),
	}

	log.Printf("vortex listening on %s", listenAddr)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	cancelWatch()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
