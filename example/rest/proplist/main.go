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

	// Print with proplist — only return address and interface fields
	result, err := client.Print(ctx, "ip/address",
		rest.WithProplist("address", "interface"),
	)
	if err != nil {
		log.Fatal(err)
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}
