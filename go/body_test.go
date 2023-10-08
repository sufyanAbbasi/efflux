package main

import (
	"context"
	"testing"
)

func TestBodyGeneration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	GenerateBody(ctx)
}
