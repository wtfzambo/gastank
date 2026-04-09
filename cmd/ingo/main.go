package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"ingo/internal/auth"
	"ingo/internal/providers/copilot"
	"ingo/internal/usage"
)

func main() {
	store := auth.NewStore()

	credsPath, err := auth.DefaultCredentialsPath()
	if err != nil {
		log.Printf("ingo: could not resolve credentials path: %v", err)
	} else {
		if err := store.Load(credsPath); err != nil {
			log.Printf("ingo: could not load credentials: %v", err)
		}
	}

	service := usage.NewService(
		copilot.NewProvider(copilot.Config{CredStore: store}),
	)

	if len(os.Args) < 2 || os.Args[1] != "usage" {
		printUsage(service)
		os.Exit(1)
	}

	providerName := copilot.ProviderName
	if len(os.Args) > 2 {
		providerName = os.Args[2]
	}

	report, err := service.Fetch(context.Background(), providerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ingo: %v\n", err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "ingo: encode report: %v\n", err)
		os.Exit(1)
	}
}

func printUsage(service *usage.Service) {
	fmt.Fprintf(os.Stderr, "Usage: ingo usage [%s]\n", copilot.ProviderName)
	fmt.Fprintf(os.Stderr, "Available providers: %s\n", strings.Join(service.Providers(), ", "))
}
