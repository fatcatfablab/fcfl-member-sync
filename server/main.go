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
	"github.com/miquelruiz/fcfl-member-sync/server/userlist"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	port = flag.Int("port", 50051, "The server port")
	crt  = flag.String("crt", "certs/server.crt", "Path to the server certificate")
	key  = flag.String("key", "certs/server.key", "Path to the server private key")
	ca   = flag.String("ca", "certs/root_ca.crt", "Path to CA root certificate")
	dsn  = flag.String("dsn", os.Getenv("DSN"), "Database DSN")
)

type server struct {
	pb.UnimplementedMembershipServer
}

func (s *server) List(ctx context.Context, _ *pb.Empty) (*pb.MemberList, error) {
	log.Print("List called")
	return userlist.List(ctx)
}

func main() {
	log.Println("Oh, hai!")
	flag.Parse()

	if err := userlist.Init(*dsn); err != nil {
		log.Fatalf("error initializing userlist module: %v", err)
	}

	cert, err := tls.LoadX509KeyPair(*crt, *key)
	if err != nil {
		log.Fatalf("error loading certs: %v", err)
	}

	certPool := x509.NewCertPool()
	caBytes, err := os.ReadFile(*ca)
	if err != nil {
		log.Fatalf("error reading %q: %v", *ca, err)
	}
	if !certPool.AppendCertsFromPEM(caBytes) {
		log.Fatalf("error adding CA to pool")
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
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
