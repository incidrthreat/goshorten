package main

import (
	"context"
	"log"
	"strconv"
	"time"

	pb "github.com/incidrthreat/goshorten/frontend-go/pb"
	"google.golang.org/grpc"
)

func main() {
	var conn *grpc.ClientConn
	conn, err := grpc.Dial("localhost:9000", grpc.WithInsecure())
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()

	c := pb.NewShortenerClient(conn)

	start := time.Now()
	i := 1

	for i <= 99999 {
		resp, err := c.CreateURL(context.Background(), &pb.ShortURLReq{LongUrl: "test" + strconv.Itoa(i)})
		if err != nil {
			log.Println(err)
		}
		log.Println(resp)
		i++
	}

	log.Printf("%d Codes generated in %v", i, time.Since(start))
}
