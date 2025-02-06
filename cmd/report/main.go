package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	ua "github.com/miquelruiz/go-unifi-access-api"
	"github.com/miquelruiz/go-unifi-access-api/schema"
)

const (
	deactivated = "DEACTIVATED"
)

var (
	uaHost  = flag.String("uaHost", os.Getenv("UA_HOST"), "Hostname or IP of the UniFi Access endpoint")
	uaToken = flag.String("token", os.Getenv("UA_TOKEN"), "Auth token for the UniFi Access API")
)

func main() {
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

	users, err := uaClient.ListUsers()
	if err != nil {
		log.Fatalf("error listing users: %s", err)
	}

	withCard := make([]schema.UserResponse, 0)
	withoutCard := make([]schema.UserResponse, 0)
	for _, u := range users {
		if u.Status == deactivated {
			continue
		}
		if len(u.NfcCards) > 0 {
			withCard = append(withCard, u)
		} else {
			withoutCard = append(withoutCard, u)
		}
	}

	fmt.Printf("Total members:        %03d\n", len(withCard)+len(withoutCard))
	fmt.Printf("Members with card:    %03d\n", len(withCard))
	fmt.Printf("Members without card: %03d\n", len(withoutCard))
}
