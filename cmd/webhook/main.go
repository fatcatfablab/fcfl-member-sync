package main

import (
	"flag"
	"os"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/db"
	"github.com/fatcatfablab/fcfl-member-sync/stripe/listener"
	"github.com/fatcatfablab/fcfl-member-sync/version"
)

var (
	stripeEndpointSecret string
	listenAddr           string
	listenEndpoint       string
	dsn                  string
	dryRun               bool
	versionflag          bool
)

func init() {
	flag.StringVar(&stripeEndpointSecret, "endpoint-secret", os.Getenv("STRIPE_ENDPOINT_SECRET"), "Stripe endpoint secret")
	flag.StringVar(&listenAddr, "listen-address", "127.0.0.1:8081", "Address to listen on")
	flag.StringVar(&listenEndpoint, "listen-endpoint", "/stripe_events", "Endpoint of the listener")
	flag.StringVar(&dsn, "dsn", os.Getenv("WEBHOOK_DSN"), "Database connection string")
	flag.BoolVar(&versionflag, "version", false, "Print the version and exit")
}

func main() {
	flag.Parse()

	if versionflag {
		version.PrintVersion()
		return
	}

	if dsn == "" {
		panic("No database connection string given")
	}

	d, err := db.New(dsn)
	if err != nil {
		panic(err)
	}

	l := listener.New(stripeEndpointSecret, listenAddr, listenEndpoint, d)
	if err := l.Start(); err != nil {
		panic(err)
	}
}
