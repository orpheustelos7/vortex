# Vortex — Distributed Rate Limiter & API Gateway

Vortex is a distributed API gateway in Go that enforces tenant-aware token-bucket quotas globally using Redis, with dynamic policy updates from etcd.

## Features

- Reverse proxy routing to tenant backends
- Per-tenant global distributed token-bucket rate limiting (Redis Lua)
- API key and JWT (HS256) authentication
- Dynamic policy hot reload from etcd watch events
- Prometheus metrics and health probe endpoints
- Docker, Kubernetes manifests, and GitHub Actions CI

## Architecture

Request path:

1. Authenticate (`X-API-Key` or `Authorization: Bearer`)
2. Load tenant policy from in-memory cache (`X-Tenant-ID`)
3. Enforce distributed rate limit via Redis Lua script
4. Proxy request to configured backend
5. Emit metrics

Configuration path:

- Policies stored under etcd prefix: `/vortex/policies/<tenant_id>`
- Each policy update is watched and applied in memory without restart

## Policy format

```json
{
  "tenant_id": "tenant-a",
  "rate": 1000,
  "burst": 2000,
  "backend_url": "http://tenant-a-service.default.svc.cluster.local",
  "api_keys": ["k1", "k2"],
  "jwt_secret": "super-secret"
}
```

## Local run

```bash
docker compose up --build
```

## Seed a policy in etcd

```bash
docker exec -it $(docker ps -qf name=etcd) etcdctl \
  --endpoints=http://localhost:2379 \
  put /vortex/policies/tenant-a '{"tenant_id":"tenant-a","rate":100,"burst":200,"backend_url":"http://httpbin.org","api_keys":["demo-key"]}'
```

## Endpoints

- `GET /healthz`
- `GET /metrics`
- `/*` proxied to tenant backend

Required request header:

- `X-Tenant-ID: <tenant>`

## Build and test

```bash
go test ./...
```
