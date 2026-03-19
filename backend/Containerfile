FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG APP_VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${APP_VERSION}" -o grpc-server .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o healthcheck ./healthcheck

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/grpc-server .
COPY --from=builder /app/healthcheck .
COPY --from=builder /app/migrations/ ./migrations/
COPY --from=builder /app/pb/url_service.swagger.json ./pb/
COPY --from=builder /app/config.json.example ./config.json

EXPOSE 9000 8080

CMD ["./grpc-server"]
