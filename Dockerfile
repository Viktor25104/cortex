# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
RUN apk add --no-cache libpcap-dev build-base
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o cortex ./main.go

FROM alpine:latest
RUN apk add --no-cache libpcap
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /opt/cortex
COPY --from=builder /src/cortex ./cortex
COPY --from=builder /src/nmap-service-probes ./nmap-service-probes
COPY --from=builder /src/.env ./.env
RUN chown -R appuser:appgroup /opt/cortex
USER appuser
EXPOSE 8080
ENV REDIS_ADDR=cortex-redis:6379
ENTRYPOINT ["./cortex", "--server"]
