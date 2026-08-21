FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/upstream main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
ENV TZ=Asia/Shanghai
COPY --from=builder /out/upstream /usr/local/bin/upstream
EXPOSE 9001
USER nobody
ENTRYPOINT ["upstream"]
