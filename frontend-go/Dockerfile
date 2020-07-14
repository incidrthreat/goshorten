FROM golang:1.14-alpine

RUN mkdir /app

WORKDIR /app/

COPY . /app/

RUN go build -o grpc-client ./cmd/*

EXPOSE 8081

CMD ["./grpc-client"]