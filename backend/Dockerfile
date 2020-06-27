FROM golang:1.14-alpine

RUN mkdir /app

WORKDIR /app/

COPY . /app/

RUN go build -o grpc-server .

EXPOSE 8080

CMD ["./grpc-server"]