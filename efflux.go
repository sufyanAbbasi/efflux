package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	MakeBaseImage().Download()
	ctx, cancel := context.WithCancel(context.Background())
	// Setting up signal capturing
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	fs := http.FileServer(http.Dir("./public"))
	server := &http.Server{Addr: ":3000", Handler: fs}

	go func() {
		log.Print("Listening on :3000...")
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
	GenerateBody(ctx)

	// Waiting for SIGINT (kill -2)
	select {
	case <-signalChan: // first signal, cancel context
		cancel()
	case <-ctx.Done():
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			panic(err) // failure/timeout shutting down the server gracefully
		}
	}
	<-signalChan // second signal, hard exit
	os.Exit(2)
}
