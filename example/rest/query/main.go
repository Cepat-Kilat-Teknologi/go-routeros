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

	// Run with query — find interfaces that are ether OR vlan type
	result, err := client.Run(ctx, "interface/print", nil,
		rest.WithProplist("name", "type"),
		rest.WithQuery("type=ether", "type=vlan", "#|"),
	)
	if err != nil {
		log.Fatal(err)
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}
