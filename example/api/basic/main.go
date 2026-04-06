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

	// Auth
	reply, err := client.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Platform:", reply.Re[0].Map["platform"])

	// Print IP addresses
	reply, err = client.Print(ctx, "/ip/address",
		api.WithProplist("address", "interface"),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, re := range reply.Re {
		fmt.Printf("  %s on %s\n", re.Map["address"], re.Map["interface"])
	}

	// Add
	reply, err = client.Add(ctx, "/ip/address", map[string]string{
		"address":   "10.0.0.1/24",
		"interface": "ether1",
	})
	if err != nil {
		log.Fatal(err)
	}
	id, _ := reply.Done.Get("ret")
	fmt.Println("Added:", id)

	// Remove
	_, err = client.Remove(ctx, "/ip/address", id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Removed:", id)
}
