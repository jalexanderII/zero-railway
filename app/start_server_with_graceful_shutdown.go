package app

import (
	"log"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v2"
)

// StartServerWithGracefulShutdown function for starting server with a graceful shutdown.
func StartServerWithGracefulShutdown(a *fiber.App, port string) {
	// Create tls certificate
	// cer, err := tls.LoadX509KeyPair("certs/ssl.cert", "certs/ssl.key")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// config := &tls.Config{Certificates: []tls.Certificate{cer}}

	// Create a channel for idle connections.
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt) // Catch OS signals.
		<-sigint

		// Received an interrupt signal, shutdown.
		if err := a.Shutdown(); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("Oops... Server is not shutting down! Reason: %v", err)
		}

		close(idleConnsClosed)
	}()

	// Create custom listener
	// ln, err := tls.Listen("tcp", ":8080", config)
	// if err != nil {
	// 	panic(err)
	// }

	// Run server.
	if err := a.Listen(port); err != nil {
		log.Printf("Oops... Server is not running! Reason: %v", err)
	}
	<-idleConnsClosed
}
