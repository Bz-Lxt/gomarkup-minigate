FROM golang:1.23-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod ./
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/minigate ./cmd/minigate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 10001 minigate
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /out/minigate /app/minigate
USER minigate
EXPOSE 8080 8081
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD wget -qO- http://127.0.0.1:8081/api/v1/health || exit 1
ENTRYPOINT ["/app/minigate"]
