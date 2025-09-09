# syntax=docker/dockerfile:1.7
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git build-base ca-certificates && update-ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go mod tidy

# ターゲットの OS/ARCH を buildx に合わせる（固定せずに自動追従）
ARG TARGETOS=linux
ARG TARGETARCH=arm64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -tags osusergo,netgo -ldflags='-s -w -extldflags "-static"' \
    -o /out/api ./cmd/api
# WORKER
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH  \
    go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# distroless の static 版（完全静的バイナリ向け）
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /
COPY --from=build /out/api /api
COPY --from=build /out/worker /worker
# ✨ HTTPS が必要な場合のために CA 証明書を同梱
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]

