package main

import (
	"context"
	"log"
	"strconv"
	"time"

	pb "github.com/incidrthreat/goshorten/frontend-go/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	var conn *grpc.ClientConn
	clientCert, err := credentials.NewClientTLSFromFile("server.crt", "")
	if err != nil {
		log.Fatal("Couldnt find the file.")
	}

	conn, err = grpc.DialContext(context.Background(), "localhost:9000", grpc.WithTransportCredentials(clientCert))
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()

	c := pb.NewShortenerClient(conn)

	start := time.Now()
	i := 1

	for i <= 9999 {
		resp, err := c.CreateURL(context.Background(), &pb.URL{LongUrl: "test" + strconv.Itoa(i)})
		if err != nil {
			log.Println(err)
		}
		log.Println(resp)
		i++
	}

	log.Printf("%d Codes generated in %v", i, time.Since(start))
}
