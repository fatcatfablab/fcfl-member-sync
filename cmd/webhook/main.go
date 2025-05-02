package main

import (
	"flag"
	"os"

	"github.com/fatcatfablab/fcfl-member-sync/stripe/db"
	"github.com/fatcatfablab/fcfl-member-sync/stripe/listener"
	_ "github.com/go-sql-driver/mysql"
)

var (
	stripeEndpointSecret string
	listenAddr           string
	listenEndpoint       string
	dsn                  string
	dryRun               bool
)

func init() {
	flag.StringVar(&stripeEndpointSecret, "endpoint-secret", os.Getenv("STRIPE_ENDPOINT_SECRET"), "Stripe endpoint secret")
	flag.StringVar(&listenAddr, "listen-address", "127.0.0.1:8081", "Address to listen on")
	flag.StringVar(&listenEndpoint, "listen-endpoint", "/stripe_events", "Endpoint of the listener")
	flag.StringVar(&dsn, "dsn", os.Getenv("WEBHOOK_DSN"), "Database connection string")
	flag.BoolVar(&dryRun, "dry-run", false, "Dry-run mode")
}

func main() {
	flag.Parse()
	if dsn == "" {
		panic("No database connection string given")
	}

	d, err := db.New(dsn, dryRun)
	if err != nil {
		panic(err)
	}

	l := listener.New(stripeEndpointSecret, listenAddr, listenEndpoint, d)
	if err := l.Start(); err != nil {
		panic(err)
	}
}
