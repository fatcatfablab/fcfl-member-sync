package main

import (
	"context"
	"flag"
	"log"
	"time"

	pb "github.com/miquelruiz/fcfl-member-sync/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr = flag.String("addr", "localhost:50051", "address to connect to")
)

func main() {
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("couldn't connect: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	c := pb.NewMembershipClient(conn)
	r, err := c.List(ctx, &pb.Empty{})
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	log.Printf("%v", r.GetMembers())
}
