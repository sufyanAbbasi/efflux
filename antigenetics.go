package main

import (
	"context"
	"os"
	"os/signal"
)

func main() {

	GenerateBody(context.Background())

	// Setting up signal capturing
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Waiting for SIGINT (kill -2)
	<-stop
}
