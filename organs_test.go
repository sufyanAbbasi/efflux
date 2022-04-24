package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type TestCell struct {
	*Cell
}

func (t *TestCell) Work(ctx context.Context, request Work) Work {
	fmt.Printf("Received: %v\n", request)
	request.status = 200
	request.result = "Completed."
	return request
}

func TestTwoNodeInteraction(t *testing.T) {
	testGraph := &Graph{
		allNodes: make(map[string]*Node),
	}
	ctx := context.Background()
	node1 := InitializeNewNode(ctx, testGraph, "node1")
	node2 := InitializeNewNode(ctx, testGraph, "node2")
	ConnectNodes(ctx, node1, node2)

	testCell1 := &TestCell{
		Cell: &Cell{
			workType: 12345,
		},
	}
	testCell2 := &TestCell{
		Cell: &Cell{
			workType: 6789,
		},
	}
	node1.AddWorker(testCell1)
	node2.AddWorker(testCell2)

	// Signal availability.
	node1.MakeAvailable(testCell1)
	node2.MakeAvailable(testCell2)

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
	node1.MakeAvailable(testCell1)
	node2.MakeAvailable(testCell2)

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
	node2 := InitializeNewNode(ctx, testGraph, "node2")
	node3 := InitializeNewNode(ctx, testGraph, "node3")
	ConnectNodes(ctx, node1, node2)
	ConnectNodes(ctx, node1, node3)

	testCell1 := &TestCell{
		Cell: &Cell{
			workType: 12345,
		},
	}
	testCell2 := &TestCell{
		Cell: &Cell{
			workType: 12345,
		},
	}
	testCell3 := &TestCell{
		Cell: &Cell{
			workType: 12345,
		},
	}
	node1.AddWorker(testCell1)
	node2.AddWorker(testCell2)
	node3.AddWorker(testCell3)

	node1.MakeAvailable(testCell1)
	node2.MakeAvailable(testCell2)
	node3.MakeAvailable(testCell3)

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
