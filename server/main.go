package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/miquelruiz/fcfl-member-sync/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	cert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
	if err != nil {
		log.Fatalf("error loading certs: %v", err)
	}

	ca := x509.NewCertPool()
	caFilePath := "certs/root_ca.crt"
	caBytes, err := os.ReadFile(caFilePath)
	if err != nil {
		log.Fatalf("error reading %q: %v", caFilePath, err)
	}
	if !ca.AppendCertsFromPEM(caBytes) {
		log.Fatalf("error adding CA to pool")
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    ca,
	}

	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	pb.RegisterMembershipServer(s, &server{})

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("couldn't listen: %v", err)
	}

	log.Printf("server listening on %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("error serving: %v", err)
	}
}
