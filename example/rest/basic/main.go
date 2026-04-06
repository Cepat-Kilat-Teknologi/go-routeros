package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/rest"
)

func main() {
	client := rest.NewClient("192.168.88.1", "admin", "",
		rest.WithInsecureSkipVerify(true),
	)
	ctx := context.Background()

	// Auth
	info, err := client.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Auth OK:", info)

	// Print
	addresses, err := client.Print(ctx, "ip/address")
	if err != nil {
		log.Fatal(err)
	}
	data, _ := json.MarshalIndent(addresses, "", "  ")
	fmt.Println("IP Addresses:", string(data))

	// Add
	payload, _ := json.Marshal(map[string]string{
		"address":   "10.0.0.1/24",
		"interface": "ether1",
	})
	added, err := client.Add(ctx, "ip/address", payload)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added:", added)

	// Remove (use the .id from the Add response)
	if m, ok := added.(map[string]interface{}); ok {
		if id, ok := m[".id"].(string); ok {
			_, err = client.Remove(ctx, "ip/address/"+id)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Removed:", id)
		}
	}
}
