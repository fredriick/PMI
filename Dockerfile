# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /proxy-mesh .

# Final stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /proxy-mesh /app/proxy-mesh
COPY config.yaml /app/config.yaml

RUN addgroup -g 1000 -S proxymesh && \
    adduser -u 1000 -S proxymesh -G proxymesh && \
    chown -R proxymesh:proxymesh /app

USER proxymesh

EXPOSE 8000 9000

CMD ["/app/proxy-mesh"]
