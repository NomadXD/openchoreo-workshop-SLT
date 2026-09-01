# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/chat-gateway .

FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/chat-gateway /app/chat-gateway
USER app

EXPOSE 8080
ENTRYPOINT ["/app/chat-gateway"]
