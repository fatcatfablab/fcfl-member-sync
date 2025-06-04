package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"

	ua "github.com/miquelruiz/go-unifi-access-api"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/db"
	"github.com/fatcatfablab/fcfl-member-sync/stripe/listener"
	"github.com/fatcatfablab/fcfl-member-sync/updater"
	"github.com/fatcatfablab/fcfl-member-sync/version"
)

var (
	stripeEndpointSecret string
	listenAddr           string
	listenEndpoint       string
	dsn                  string
	uaToken              string
	uaHost               string
	dryRun               bool
	versionflag          bool
)

func init() {
	flag.StringVar(&stripeEndpointSecret, "endpoint-secret", os.Getenv("STRIPE_ENDPOINT_SECRET"), "Stripe endpoint secret")
	flag.StringVar(&listenAddr, "listen-address", "127.0.0.1:8081", "Address to listen on")
	flag.StringVar(&listenEndpoint, "listen-endpoint", "/stripe_events", "Endpoint of the listener")
	flag.StringVar(&dsn, "dsn", os.Getenv("WEBHOOK_DSN"), "Database connection string")
	flag.StringVar(&uaToken, "uaToken", os.Getenv("UA_TOKEN"), "UniFi Access token")
	flag.StringVar(&uaHost, "uaHost", "https://192.168.2.1:12445", "UniFi Access url")
	flag.BoolVar(&versionflag, "version", false, "Print the version and exit")
	flag.BoolVar(&dryRun, "dry-run", false, "Wether UniFi Access should be called or not")
}

func main() {
	flag.Parse()

	if versionflag {
		version.PrintVersion()
		return
	}

	if dsn == "" {
		log.Fatal("No database connection string given")
	}

	if uaToken == "" {
		log.Fatal("No UniFi Access token given")
	}

	uaClient, err := ua.NewWithHttpClient(uaHost, uaToken, &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})
	if err != nil {
		log.Fatalf("error creating UniFi Access API client: %s", err)
	}
	uniFiUpdater := updater.New(uaClient, dryRun)

	d, err := db.New(dsn)
	if err != nil {
		log.Fatalf("error connecting to database: %s", err)
	}

	l := listener.New(stripeEndpointSecret, listenAddr, listenEndpoint, d, uniFiUpdater)
	if err := l.Start(); err != nil {
		log.Fatalf("error after calling listener.Start: %s", err)
	}
}
