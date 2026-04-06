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

	// Add a new IP address
	reply, err := client.Add(ctx, "/ip/address", map[string]string{
		"address":   "10.0.0.1/24",
		"interface": "ether1",
		"comment":   "Added via API",
	})
	if err != nil {
		log.Fatal(err)
	}

	id, _ := reply.Done.Get("ret")
	fmt.Println("Added record ID:", id)

	// Update the record
	_, err = client.Set(ctx, "/ip/address", map[string]string{
		".id":     id,
		"comment": "Updated via API",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Updated record:", id)

	// Verify the update
	reply, err = client.Print(ctx, "/ip/address",
		api.WithProplist(".id", "address", "comment"),
		api.WithQuery("?=.id="+id),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, re := range reply.Re {
		fmt.Printf("ID: %s, Address: %s, Comment: %s\n",
			re.Map[".id"], re.Map["address"], re.Map["comment"])
	}

	// Clean up — remove the record
	_, err = client.Remove(ctx, "/ip/address", id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Removed record:", id)
}
