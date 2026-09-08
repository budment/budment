package planner

import (
	"testing"
	"time"

	pb "github.com/vunas/blaster/internal/planner/pb"
)

func TestGraphCompiler_CompileBasicNodes(t *testing.T) {
	compiler := NewGraphCompiler()

	// Construct Protobuf scenario with a SleepNode and an ActionNode containing a Before pipeline
	pbScenario := &pb.Scenario{
		Name: "Smoke Scenario",
		Execution: &pb.Pipeline{
			Steps: []*pb.Node{
				{
					Id: "sleep_step",
					Type: &pb.Node_Sleep{
						Sleep: &pb.SleepNode{DurationS: 2.5},
					},
				},
				{
					Id: "action_step",
					Type: &pb.Node_Action{
						Action: &pb.ActionNode{
							Protocol: "HTTP",
							Method:   "GET",
							Target:   "https://api.example.com/users",
							Before: &pb.Pipeline{
								Steps: []*pb.Node{
									{
										Id: "log_step",
										Type: &pb.Node_Log{
											Log: &pb.LogNode{Message: "Preparing request"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	graph, err := compiler.Compile(pbScenario)
	if err != nil {
		t.Fatalf("compilation failed unexpectedly: %v", err)
	}

	if len(graph.Execution) != 2 {
		t.Fatalf("executable nodes count mismatch: expected 2, got %d", len(graph.Execution))
	}

	// Verify SleepNode
	sleepNode, ok := graph.Execution[0].(*SleepNode)
	if !ok {
		t.Fatalf("node 0 type mismatch: expected *SleepNode, got %T", graph.Execution[0])
	}
	expectedDur := time.Duration(2.5 * float64(time.Second))
	if sleepNode.Duration != expectedDur {
		t.Errorf("duration mismatch: expected %v, got %v", expectedDur, sleepNode.Duration)
	}

	// Verify ActionNode and nested BeforePipeline
	actionNode, ok := graph.Execution[1].(*ActionNode)
	if !ok {
		t.Fatalf("node 1 type mismatch: expected *ActionNode, got %T", graph.Execution[1])
	}
	if actionNode.Protocol != "HTTP" || actionNode.Method != "GET" {
		t.Errorf("ActionNode configuration mismatch: protocol=%s method=%s", actionNode.Protocol, actionNode.Method)
	}
	if len(actionNode.BeforePipeline) != 1 {
		t.Fatalf("BeforePipeline length mismatch: expected 1, got %d", len(actionNode.BeforePipeline))
	}
}

func TestGraphCompiler_DepthLimitExceeded(t *testing.T) {
	compiler := NewGraphCompiler()

	// Build recursive pipeline exceeding the 50-tier depth limit
	currentPipeline := &pb.Pipeline{}
	rootPipeline := currentPipeline

	for i := 0; i < 55; i++ {
		nextPipeline := &pb.Pipeline{}
		currentPipeline.Steps = append(currentPipeline.Steps, &pb.Node{
			Id: "nested_branch",
			Type: &pb.Node_Branch{
				Branch: &pb.BranchNode{
					TruePath: nextPipeline,
				},
			},
		})
		currentPipeline = nextPipeline
	}

	pbScenario := &pb.Scenario{
		Execution: rootPipeline,
	}

	_, err := compiler.Compile(pbScenario)
	if err == nil {
		t.Fatal("expected depth limit error, got nil")
	}
}
