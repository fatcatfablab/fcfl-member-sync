//go:build stdout

package sync

import (
	"log"

	pb "github.com/miquelruiz/fcfl-member-sync/proto"
)

func Sync(members []*pb.Member) {
	log.Printf("Members: %d", len(members))
	for _, member := range members {
		log.Printf("%v", member)
	}
}
