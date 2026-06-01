FROM golang:1.22 AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/vortex ./cmd/vortex

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/vortex /vortex
EXPOSE 8080
ENTRYPOINT ["/vortex"]
