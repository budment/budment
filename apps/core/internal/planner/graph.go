package planner

import (
	"errors"

	pb "github.com/vunas/blaster/internal/planner/pb"
)

type ExtractInstruction struct {
	StackID      string
	Path         string
	TotalRef     int
	IsDistribute bool
}

type InjectInstruction struct {
	StackID          string
	Target           string
	IsFromDistribute bool
}

type Graph struct {
	Setup     []ExecutableNode
	Execution []ExecutableNode
}

type ExecutableNode interface {
	NodeID() string
	Type() string
}

type HttpNode struct {
	ID           string
	Method       string
	URL          string
	BeforeHookID string
	AfterHookID  string

	Extracts []ExtractInstruction
	Injects  []InjectInstruction
}

func (n *HttpNode) NodeID() string { return n.ID }
func (n *HttpNode) Type() string   { return "HTTP" }

type BranchNode struct {
	ID              string
	ConditionHookID string
	TruePath        []ExecutableNode
	FalsePath       []ExecutableNode
}

func (n *BranchNode) NodeID() string { return n.ID }
func (n *BranchNode) Type() string   { return "BRANCH" }

type LoopNode struct {
	ID    string
	Count int32
	Logic []ExecutableNode
}

func (n *LoopNode) NodeID() string { return n.ID }
func (n *LoopNode) Type() string   { return "LOOP" }

type MatchNode struct {
	ID              string
	ConditionHookID string
	Cases           map[string][]ExecutableNode
	DefaultPath     []ExecutableNode
}

func (n *MatchNode) NodeID() string { return n.ID }
func (n *MatchNode) Type() string   { return "MATCH" }

type PollNode struct {
	ID              string
	ConditionHookID string
	Logic           []ExecutableNode
	Interval        string
	Timeout         string
	MaxAttempts     int32
}

func (n *PollNode) NodeID() string { return n.ID }
func (n *PollNode) Type() string   { return "POLL" }

type ScriptNode struct {
	ID     string
	HookID string
}

func (n *ScriptNode) NodeID() string { return n.ID }
func (n *ScriptNode) Type() string   { return "SCRIPT" }

type GraphCompiler struct{}

func NewGraphCompiler() *GraphCompiler { return &GraphCompiler{} }

func (c *GraphCompiler) Compile(scenario *pb.Scenario) (*Graph, error) {
	graph := &Graph{}
	if scenario.Setup != nil {
		nodes, _ := c.compilePipeline(scenario.Setup, 0)
		graph.Setup = nodes
	}
	if scenario.Execution != nil {
		nodes, _ := c.compilePipeline(scenario.Execution, 0)
		graph.Execution = nodes
	}
	return graph, nil
}

func (c *GraphCompiler) compilePipeline(pipeline *pb.Pipeline, depth int) ([]ExecutableNode, error) {
	if depth > 50 {
		return nil, errors.New("pipeline depth exceeded maximum limit")
	}

	var nodes []ExecutableNode
	if pipeline == nil {
		return nodes, nil
	}

	for _, step := range pipeline.Steps {
		switch n := step.Type.(type) {
		case *pb.Node_Http:
			nodes = append(nodes, &HttpNode{
				ID:           step.Id,
				Method:       n.Http.Method,
				URL:          n.Http.Url,
				BeforeHookID: n.Http.BeforeHookId,
				AfterHookID:  n.Http.AfterHookId,
			})
		case *pb.Node_Branch:
			truePath, _ := c.compilePipeline(n.Branch.TruePath, depth+1)
			falsePath, _ := c.compilePipeline(n.Branch.FalsePath, depth+1)
			nodes = append(nodes, &BranchNode{
				ID:              step.Id,
				ConditionHookID: n.Branch.ConditionHookId,
				TruePath:        truePath,
				FalsePath:       falsePath,
			})
		case *pb.Node_Loop:
			var count int32 = 1
			if loopCount, ok := n.Loop.Config.(*pb.LoopNode_Count); ok {
				count = loopCount.Count
			}
			logicPath, _ := c.compilePipeline(n.Loop.Logic, depth+1)
			nodes = append(nodes, &LoopNode{ID: step.Id, Count: count, Logic: logicPath})
		case *pb.Node_Match:
			cases := make(map[string][]ExecutableNode)
			for k, v := range n.Match.Cases {
				cases[k], _ = c.compilePipeline(v, depth+1)
			}
			defaultPath, _ := c.compilePipeline(n.Match.DefaultPath, depth+1)
			nodes = append(nodes, &MatchNode{ID: step.Id, ConditionHookID: n.Match.ConditionHookId, Cases: cases, DefaultPath: defaultPath})
		case *pb.Node_Poll:
			var interval, timeout string = "1s", "30s"
			var maxAttempts int32 = 60
			if n.Poll.Policy != nil {
				if n.Poll.Policy.Interval != "" {
					interval = n.Poll.Policy.Interval
				}
				if n.Poll.Policy.Timeout != "" {
					timeout = n.Poll.Policy.Timeout
				}
				if n.Poll.Policy.MaxAttempts > 0 {
					maxAttempts = n.Poll.Policy.MaxAttempts
				}
			}
			logicPath, _ := c.compilePipeline(n.Poll.Logic, depth+1)
			nodes = append(nodes, &PollNode{ID: step.Id, ConditionHookID: n.Poll.ConditionHookId, Logic: logicPath, Interval: interval, Timeout: timeout, MaxAttempts: maxAttempts})
		case *pb.Node_Script:
			nodes = append(nodes, &ScriptNode{ID: step.Id, HookID: n.Script.HookId})
		}
	}
	return nodes, nil
}
