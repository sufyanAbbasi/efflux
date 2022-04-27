package main

import (
	"context"
	"os"
	"os/signal"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	GenerateBody(ctx)

	// Setting up signal capturing
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Waiting for SIGINT (kill -2)
	<-stop
}
