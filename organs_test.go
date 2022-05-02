package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type TestWorker struct {
	*Cell
}

func TestSelfNodeInteraction(t *testing.T) {
	testGraph := &Graph{
		allNodes: make(map[string]*Node),
	}
	ctx := context.Background()
	node1 := InitializeNewNode(ctx, testGraph, "node1")
	node1.materialPool = nil
	ConnectNodes(ctx, node1, node1)

	testWorker1 := &TestWorker{
		Cell: &Cell{
			workType: 12345,
		},
	}
	node1.AddWorker(testWorker1)

	// Signal availability.
	node1.MakeAvailable(testWorker1)

	result := node1.RequestWork(Work{
		workType: 12345,
	})

	if result.workType != 12345 {
		t.Errorf("Expected workType to be 12345, got: %v", result.workType)
	}
	if result.result != "Completed." {
		t.Errorf("Expected result to be \"Completed.\", got: %v", result.result)
	}
	if result.status != 200 {
		t.Errorf("Expected status to be 200, got: %v", result.status)
	}
}

func TestTwoNodeInteraction(t *testing.T) {
	testGraph := &Graph{
		allNodes: make(map[string]*Node),
	}
	ctx := context.Background()
	node1 := InitializeNewNode(ctx, testGraph, "node1")
	node1.materialPool = nil
	node2 := InitializeNewNode(ctx, testGraph, "node2")
	node2.materialPool = nil
	ConnectNodes(ctx, node1, node2)

	testWorker1 := &TestWorker{
		Cell: &Cell{
			workType: 12345,
		},
	}
	testWorker2 := &TestWorker{
		Cell: &Cell{
			workType: 6789,
		},
	}
	node1.AddWorker(testWorker1)
	node2.AddWorker(testWorker2)

	// Signal availability.
	node1.MakeAvailable(testWorker1)
	node2.MakeAvailable(testWorker2)

	result := node1.RequestWork(Work{
		workType: 6789,
	})

	if result.workType != 6789 {
		t.Errorf("Expected workType to be 6789, got: %v", result.workType)
	}
	if result.result != "Completed." {
		t.Errorf("Expected result to be \"Completed.\", got: %v", result.result)
	}
	if result.status != 200 {
		t.Errorf("Expected status to be 200, got: %v", result.status)
	}

	fmt.Println()
	// Signal availability again.
	node1.MakeAvailable(testWorker1)
	node2.MakeAvailable(testWorker2)

	result = node2.RequestWork(Work{
		workType: 12345,
	})

	if result.workType != 12345 {
		t.Errorf("Expected workType to be 12345, got: %v", result.workType)
	}
	if result.result != "Completed." {
		t.Errorf("Expected result to be \"Completed.\", got: %v", result.result)
	}
	if result.status != 200 {
		t.Errorf("Expected status to be 200, got: %v", result.status)
	}
}

func TestThreeNodeInteraction(t *testing.T) {
	testGraph := &Graph{
		allNodes: make(map[string]*Node),
	}
	ctx := context.Background()
	node1 := InitializeNewNode(ctx, testGraph, "node1")
	node1.materialPool = nil
	node2 := InitializeNewNode(ctx, testGraph, "node2")
	node2.materialPool = nil
	node3 := InitializeNewNode(ctx, testGraph, "node3")
	node3.materialPool = nil
	ConnectNodes(ctx, node1, node2)
	ConnectNodes(ctx, node1, node3)

	testWorker1 := &TestWorker{
		Cell: &Cell{
			workType: 12345,
		},
	}
	testWorker2 := &TestWorker{
		Cell: &Cell{
			workType: 12345,
		},
	}
	TestWorker3 := &TestWorker{
		Cell: &Cell{
			workType: 12345,
		},
	}
	node1.AddWorker(testWorker1)
	node2.AddWorker(testWorker2)
	node3.AddWorker(TestWorker3)

	node1.MakeAvailable(testWorker1)
	node2.MakeAvailable(testWorker2)
	node3.MakeAvailable(TestWorker3)

	result := node1.RequestWork(Work{
		workType: 12345,
	})

	if result.workType != 12345 {
		t.Errorf("Expected workType to be 12345, got: %v", result.workType)
	}
	if result.result != "Completed." {
		t.Errorf("Expected result to be \"Completed.\", got: %v", result.result)
	}
	if result.status != 200 {
		t.Errorf("Expected status to be 200, got: %v", result.status)
	}

	// Don't need to signal availability again, since the result channel
	// has extra work done on the buffer.
	fmt.Println()

	// Sleep to settle other threads
	time.Sleep(time.Duration(100 * time.Millisecond))

	result = node1.RequestWork(Work{
		workType: 12345,
	})

	if result.workType != 12345 {
		t.Errorf("Expected workType to be 12345, got: %v", result.workType)
	}
	if result.result != "Completed." {
		t.Errorf("Expected result to be \"Completed.\", got: %v", result.result)
	}
	if result.status != 200 {
		t.Errorf("Expected status to be 200, got: %v", result.status)
	}
}
