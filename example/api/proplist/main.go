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

	// Print with proplist — only return specific fields
	// This improves performance by skipping slow-access properties
	reply, err := client.Print(ctx, "/ip/address",
		api.WithProplist("address", "interface", "disabled"),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, re := range reply.Re {
		fmt.Printf("Address: %s, Interface: %s, Disabled: %s\n",
			re.Map["address"], re.Map["interface"], re.Map["disabled"])
	}
}
