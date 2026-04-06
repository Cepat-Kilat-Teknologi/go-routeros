package api_test

import (
	"context"
	"fmt"
	"log"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api"
)

func ExampleDial() {
	client, err := api.Dial("192.168.88.1", "admin", "")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	reply, err := client.Print(context.Background(), "/ip/address",
		api.WithProplist("address", "interface"),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, re := range reply.Re {
		fmt.Printf("%s on %s\n", re.Map["address"], re.Map["interface"])
	}
}

func ExampleClient_Print() {
	client, err := api.Dial("192.168.88.1", "admin", "")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Print with query: find ether OR vlan interfaces
	reply, err := client.Print(context.Background(), "/interface",
		api.WithProplist("name", "type"),
		api.WithQuery("?type=ether", "?type=vlan", "?#|"),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, re := range reply.Re {
		fmt.Printf("%s (%s)\n", re.Map["name"], re.Map["type"])
	}
}

func ExampleDeviceError() {
	client, err := api.Dial("192.168.88.1", "admin", "")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	_, err = client.Print(context.Background(), "/nonexistent")
	if err != nil {
		if de, ok := err.(*api.DeviceError); ok {
			fmt.Printf("Category: %d, Message: %s\n", de.Category, de.Message)
		}
	}
}
