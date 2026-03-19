FROM node:22-alpine AS frontend

WORKDIR /build

COPY ui-src/package.json ui-src/package-lock.json* ./
RUN npm install

COPY ui-src/ .

# Build outputs to /build/../dist = /dist (we override outDir)
RUN npx vite build --outDir /dist

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG APP_VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${APP_VERSION}" -o grpc-client ./cmd/*
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o healthcheck ./healthcheck

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/grpc-client .
COPY --from=builder /app/healthcheck .
COPY --from=frontend /dist/ ./dist/

EXPOSE 8081

CMD ["./grpc-client"]
