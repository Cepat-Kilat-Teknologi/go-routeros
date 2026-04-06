package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api"
)

func main() {
	client, err := api.Dial("192.168.88.1", "admin", "")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Query: find ether OR vlan interfaces
	reply, err := client.Print(ctx, "/interface",
		api.WithProplist("name", "type"),
		api.WithQuery("?type=ether", "?type=vlan", "?#|"),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, re := range reply.Re {
		fmt.Printf("  %s (%s)\n", re.Map["name"], re.Map["type"])
	}
}
