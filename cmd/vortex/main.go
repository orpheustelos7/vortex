package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
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
	go watcher.Watch(context.Background())

	metrics := observability.NewMetrics()
	limiter := ratelimiter.NewRedisTokenBucket(redisClient)
	server := gateway.New(store, limiter, metrics)

	log.Printf("vortex listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, server.Handler()); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
