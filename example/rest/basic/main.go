package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/rest"
)

// IPAddress represents a RouterOS IP address record.
// Define your own struct to match the fields you need.
type IPAddress struct {
	ID        string `json:".id"`
	Address   string `json:"address"`
	Network   string `json:"network"`
	Interface string `json:"interface"`
	Disabled  string `json:"disabled"`
	Comment   string `json:"comment"`
}

// SystemResource represents RouterOS system resource info.
type SystemResource struct {
	BoardName string `json:"board-name"`
	Platform  string `json:"platform"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	CPULoad   string `json:"cpu-load"`
}

func main() {
	client := rest.NewClient("192.168.88.1", "admin", "",
		rest.WithInsecureSkipVerify(true),
	)
	ctx := context.Background()

	// =========================================
	// Auth — decode response to typed struct
	// =========================================
	info, err := client.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Convert interface{} to typed struct via JSON re-encode
	var resource SystemResource
	if err := decode(info, &resource); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Board: %s, Version: %s, Platform: %s\n",
		resource.BoardName, resource.Version, resource.Platform)

	// =========================================
	// Print — decode array response
	// =========================================
	result, err := client.Print(ctx, "ip/address")
	if err != nil {
		log.Fatal(err)
	}

	// Decode to slice of typed struct
	var addresses []IPAddress
	if err := decode(result, &addresses); err != nil {
		log.Fatal(err)
	}
	for _, addr := range addresses {
		fmt.Printf("  [%s] %s on %s (disabled=%s)\n",
			addr.ID, addr.Address, addr.Interface, addr.Disabled)
	}

	// =========================================
	// Add — create a new record
	// =========================================
	payload, _ := json.Marshal(map[string]string{
		"address":   "10.0.0.1/24",
		"interface": "ether1",
	})
	added, err := client.Add(ctx, "ip/address", payload)
	if err != nil {
		log.Fatal(err)
	}

	// Get the .id from the response
	var created IPAddress
	if err := decode(added, &created); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added ID:", created.ID)

	// =========================================
	// Remove — delete by ID
	// =========================================
	_, err = client.Remove(ctx, "ip/address/"+created.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Removed:", created.ID)
}

// decode converts interface{} to a typed struct via JSON re-encode/decode.
// This is the recommended pattern for working with rest package responses.
func decode(src interface{}, dst interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
