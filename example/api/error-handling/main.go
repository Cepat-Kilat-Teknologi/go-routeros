package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api"
)

func main() {
	// Connect to router
	client, err := api.Dial("192.168.88.1", "admin", "")
	if err != nil {
		log.Fatal("Connection failed:", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: Command that triggers a trap error (missing command)
	_, err = client.Print(ctx, "/nonexistent/path")
	if err != nil {
		if de, ok := err.(*api.DeviceError); ok {
			fmt.Printf("DeviceError caught!\n")
			fmt.Printf("  Category: %d\n", de.Category)
			fmt.Printf("  Message:  %s\n", de.Message)
			fmt.Printf("  Full:     %s\n", de.Error())
		} else {
			fmt.Println("Other error:", err)
		}
	}

	// Example 2: Adding with invalid parameters
	_, err = client.Add(ctx, "/ip/address", map[string]string{
		"address":   "invalid-address",
		"interface": "nonexistent",
	})
	if err != nil {
		if de, ok := err.(*api.DeviceError); ok {
			fmt.Printf("\nDeviceError caught!\n")
			fmt.Printf("  Category: %d\n", de.Category)
			fmt.Printf("  Message:  %s\n", de.Message)
		}
	}

	// Example 3: Connection error (trying to dial a non-existent host)
	_, err = api.Dial("192.168.88.254:8728", "admin", "",
		api.WithTimeout(3*1e9), // 3 seconds
	)
	if err != nil {
		fmt.Printf("\nConnection error: %s\n", err)
	}

	// Example 4: Handling fatal errors
	// Fatal errors close the connection automatically.
	// After a fatal error, you must reconnect.
	// if fe, ok := err.(*api.FatalError); ok {
	//     fmt.Println("Fatal:", fe.Message)
	//     // Must create a new client with api.Dial()
	// }

	fmt.Println("\nDone!")
}
