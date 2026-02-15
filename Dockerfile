FROM golang:1.25-alpine AS builder

RUN apk add --no-cache alsa-lib-dev gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /pink-noise ./cmd/pink-noise

FROM alpine:3.19

RUN apk add --no-cache alsa-lib

COPY --from=builder /pink-noise /usr/local/bin/pink-noise

ENV MQTT_BROKER=tcp://localhost
ENV MQTT_PORT=1883
ENV MQTT_TOPIC=homeassistant/noise
ENV SAMPLE_RATE=44100
ENV BUFFER_SIZE=2048

ENTRYPOINT ["/usr/local/bin/pink-noise"]
