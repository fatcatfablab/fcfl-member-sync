package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/fatcatfablab/fcfl-member-sync/client/sync"
	"github.com/fatcatfablab/fcfl-member-sync/client/types"
	"github.com/fatcatfablab/fcfl-member-sync/client/updater"
	pb "github.com/fatcatfablab/fcfl-member-sync/proto"
	ua "github.com/miquelruiz/go-unifi-access-api"

	"github.com/samber/lo"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	addr = flag.String("addr", os.Getenv("FCFL_CRM_ADDR"), "address to connect to")
	crt  = flag.String("crt", "certs/client.crt", "Path to the client certificate")
	key  = flag.String("key", "certs/client.key", "Path to the client private key")
	ca   = flag.String("ca", "certs/root_ca.crt", "Path to the CA root certificate")

	uaHost  = flag.String("uaHost", os.Getenv("UA_HOST"), "Hostname or IP of the UniFi Access endpoint")
	uaToken = flag.String("token", os.Getenv("UA_TOKEN"), "Auth token for the UniFi Access API")
)

func main() {
	flag.Parse()

	uaClient, err := ua.NewWithHttpClient(*uaHost, *uaToken, &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})
	if err != nil {
		log.Fatalf("error creating UniFi Access API client: %s", err)
	}
	uniFiUpdater := updater.New(uaClient)

	remoteMembers, err := getRemoteMembers()
	if err != nil {
		log.Fatalf("error getting remote members: %s", err)
	}

	localMembers, err := uniFiUpdater.List()
	if err != nil {
		log.Fatalf("error getting local members: %s", err)
	}

	if err = sync.Reconcile(remoteMembers, localMembers, uniFiUpdater); err != nil {
		log.Fatalf("error reconciling local members list: %s", err)
	}
}

func getRemoteMembers() (types.MemberSet, error) {
	conn, err := createMembershipConn()
	if err != nil {
		return nil, fmt.Errorf("couldn't connect: %w", err)
	}
	defer conn.Close()
	mClient := pb.NewMembershipClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	remoteMembers, err := mClient.List(ctx, &pb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return mapset.NewSet(lo.Map(
		remoteMembers.Members,
		func(m *pb.Member, _ int) types.ComparableMember {
			return types.ComparableMember{
				Id:        m.Id,
				FirstName: m.FirstName,
				LastName:  m.LastName,
				Status:    types.StatusActive, // remote members are always ACTIVE
			}
		},
	)...), nil
}

func createMembershipConn() (*grpc.ClientConn, error) {
	cert, err := tls.LoadX509KeyPair(*crt, *key)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert: %w", err)
	}

	certPool := x509.NewCertPool()
	caBytes, err := os.ReadFile(*ca)
	if err != nil {
		return nil, fmt.Errorf("error reading ca %q: %w", *ca, err)
	}
	if !certPool.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("failed to parse %q", *ca)
	}

	serverUrl := url.URL{Host: *addr}
	tlsConfig := &tls.Config{
		ServerName:   serverUrl.Hostname(),
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
	}

	return grpc.NewClient(serverUrl.Host, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}
