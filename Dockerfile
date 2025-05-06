FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -o /build/service_discovery ./cmd/service_discovery

FROM alpine
COPY --from=builder /build/service_discovery /usr/bin/service_discovery
ENV TZ=Europe/Moscow
CMD [ "service_discovery" ]