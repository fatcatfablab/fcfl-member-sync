package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log"
	"net/url"
	"os"
	"time"

	pb "github.com/miquelruiz/fcfl-member-sync/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	addr = flag.String("addr", "localhost:50051", "address to connect to")
	crt  = flag.String("crt", "certs/client.crt", "Path to the client certificate")
	key  = flag.String("key", "certs/client.key", "Path to the client private key")
	ca   = flag.String("ca", "certs/root_ca.crt", "Path to the CA root certificate")
)

func main() {
	flag.Parse()

	cert, err := tls.LoadX509KeyPair(*crt, *key)
	if err != nil {
		log.Fatalf("failed to load client cert: %v", err)
	}

	certPool := x509.NewCertPool()
	caBytes, err := os.ReadFile(*ca)
	if err != nil {
		log.Fatalf("error reading ca %q: %v", *ca, err)
	}
	if !certPool.AppendCertsFromPEM(caBytes) {
		log.Fatalf("failed to parse %q", *ca)
	}

	serverUrl := url.URL{Host: *addr}
	tlsConfig := &tls.Config{
		ServerName:   serverUrl.Hostname(),
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
	}

	conn, err := grpc.NewClient(serverUrl.Host, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
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
