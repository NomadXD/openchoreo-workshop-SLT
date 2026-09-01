FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/subscription-service .

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app
USER app

COPY --from=builder /out/subscription-service /usr/local/bin/subscription-service

EXPOSE 8080

CMD ["/usr/local/bin/subscription-service"]
