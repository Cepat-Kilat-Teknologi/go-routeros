package rest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/rest"
)

func ExampleNewClient() {
	client := rest.NewClient("192.168.88.1", "admin", "",
		rest.WithInsecureSkipVerify(true),
	)

	result, err := client.Print(context.Background(), "ip/address")
	if err != nil {
		log.Fatal(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func ExampleClient_Print() {
	client := rest.NewClient("192.168.88.1", "admin", "")

	// Print with proplist and filter
	result, err := client.Print(context.Background(), "ip/address",
		rest.WithProplist("address", "interface"),
		rest.WithFilter(map[string]string{"dynamic": "true"}),
	)
	if err != nil {
		log.Fatal(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func ExampleAPIError() {
	client := rest.NewClient("192.168.88.1", "admin", "")

	_, err := client.Print(context.Background(), "nonexistent")
	if err != nil {
		if apiErr, ok := err.(*rest.APIError); ok {
			fmt.Printf("Code: %d, Message: %s\n", apiErr.StatusCode, apiErr.Message)
		}
	}
}
