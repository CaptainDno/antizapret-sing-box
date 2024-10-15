package main

import (
	"fmt"
	"log"

	"github.com/CaptainDno/antizapret-sing-geosite/geosite_antizapret"
)

func main() {
	generator := geosite_antizapret.NewGenerator()

	fmt.Println("Starting....")

	if err := generator.GenerateAndWrite("output"); err != nil {
		log.Fatal(err)
	}
}
