package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/miquelruiz/fcfl-member-sync/proto"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	pb.UnimplementedMembershipServer
}

func (s *server) List(_ context.Context, _ *pb.Empty) (*pb.MemberList, error) {
	log.Print("List called")
	return nil, nil
}

func main() {
	log.Println("Oh, hai!")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("couldn't listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMembershipServer(s, &server{})
	log.Printf("server listening on %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("error serving: %v", err)
	}
}
