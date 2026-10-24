package main

import (
	_ "embed"
	"fmt"
	"os"
)

const (
	behaviour = "TEST_BEHAVIOUR"
)

//go:embed invalid-event.json
var invalidEvent string

//go:embed result-event.json
var resultEvent string

//go:embed end-event.json
var endEvent string

func main() {
	behaviour := os.Getenv(behaviour)

	switch behaviour {
	case "SUCCESS":

		fmt.Println(resultEvent)
		fmt.Println(endEvent)

		os.Exit(0)
	case "FAILURE":
		os.Exit(1)

	case "INVALID_EVENT":
		fmt.Println(invalidEvent)
		fmt.Println(endEvent)

		os.Exit(0)
	}
}
