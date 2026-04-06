package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api"
)

func main() {
	// Connect with TLS on port 8729
	// Option 1: Simple TLS (uses default port 8729)
	client, err := api.Dial("192.168.88.1", "admin", "",
		api.WithTLS(true),
		api.WithTimeout(10*time.Second),
	)
	if err != nil {
		// If the router uses a self-signed certificate, use WithTLSConfig
		fmt.Println("TLS with default config failed, trying with InsecureSkipVerify...")

		client, err = api.Dial("192.168.88.1", "admin", "",
			api.WithTLSConfig(&tls.Config{
				InsecureSkipVerify: true, // skip certificate verification
			}),
			api.WithTimeout(10*time.Second),
		)
		if err != nil {
			log.Fatal("TLS connection failed:", err)
		}
	}
	defer client.Close()

	ctx := context.Background()

	// Verify connection
	reply, err := client.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected via TLS!")
	for _, re := range reply.Re {
		fmt.Printf("  Board:    %s\n", re.Map["board-name"])
		fmt.Printf("  Version:  %s\n", re.Map["version"])
		fmt.Printf("  Platform: %s\n", re.Map["platform"])
		fmt.Printf("  Uptime:   %s\n", re.Map["uptime"])
	}
}
