package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"parentcontrol/internal/relay"
)

func main() {
	defaultPort := 9000
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			defaultPort = p
		}
	}

	defaultSecret := os.Getenv("RELAY_SECRET")

	port := flag.Int("port", defaultPort, "Port to listen on (default 9000 or $PORT)")
	secret := flag.String("secret", defaultSecret, "Global auth secret (optional, or $RELAY_SECRET)")
	flag.Parse()

	fmt.Println("==================================================")
	fmt.Println("  ParentControl Cloud Relay Server (Go + MQ Socket)")
	fmt.Println("==================================================")
	fmt.Printf("  • Listening Port   : %d\n", *port)
	if *secret != "" {
		fmt.Printf("  • Auth Secret      : Enabled (%s...)\n", (*secret)[:min(3, len(*secret))])
	} else {
		fmt.Println("  • Multi-tenant Mode: Enabled (Dynamic per-device secret)")
	}
	fmt.Println("  • WebSocket Router : /ws/router")
	fmt.Println("  • WebSocket Client : /ws/client")
	fmt.Println("  • REST API         : /api/client/*")
	fmt.Println("==================================================")

	server := relay.NewServer(*secret)
	if err := server.Start(*port); err != nil {
		log.Fatalf("[Fatal] Server terminated: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
