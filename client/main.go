package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log"
	"os"
	"time"

	pb "github.com/miquelruiz/fcfl-member-sync/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	addr = flag.String("addr", "localhost:50051", "address to connect to")
)

func main() {
	flag.Parse()

	cert, err := tls.LoadX509KeyPair("certs/client.crt", "certs/client.key")
	if err != nil {
		log.Fatalf("failed to load client cert: %v", err)
	}

	ca := x509.NewCertPool()
	caFilePath := "certs/root_ca.crt"
	caBytes, err := os.ReadFile(caFilePath)
	if err != nil {
		log.Fatalf("error reading ca %q: %v", caFilePath, err)
	}
	if !ca.AppendCertsFromPEM(caBytes) {
		log.Fatalf("failed to parse %q", caFilePath)
	}

	tlsConfig := &tls.Config{
		ServerName:   "localhost",
		Certificates: []tls.Certificate{cert},
		RootCAs:      ca,
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
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
