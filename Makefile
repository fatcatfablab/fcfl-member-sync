.PHONY: gen
gen: proto/members.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=.  --go-grpc_opt=paths=source_relative proto/members.proto

.PHONY: server
server: server/*
	go build -tags civicrm -o fcfl-member-sync-server ./server

.PHONY: client
client: client/*
	go build -o fcfl-member-sync-client ./client
