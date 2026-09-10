package planner

import (
	"errors"
	"time"

	pb "github.com/budment/budment/internal/planner/pb"
	"github.com/budment/budment/internal/template"
)

type ExecutableNode interface {
	NodeID() string
	Type() string
}

type ActionNode struct {
	ID             string
	Protocol       string
	Method         string
	Target         template.Expression
	BeforePipeline []ExecutableNode
	AfterPipeline  []ExecutableNode
}

func (n *ActionNode) NodeID() string { return n.ID }
func (n *ActionNode) Type() string   { return "ACTION" }

type BranchNode struct {
	ID              string
	ConditionHookID string
	TruePath        []ExecutableNode
	FalsePath       []ExecutableNode
}

func (n *BranchNode) NodeID() string { return n.ID }
func (n *BranchNode) Type() string   { return "BRANCH" }

type MatchNode struct {
	ID              string
	ConditionHookID string
	Cases           map[string][]ExecutableNode
	DefaultPath     []ExecutableNode
}

func (n *MatchNode) NodeID() string { return n.ID }
func (n *MatchNode) Type() string   { return "MATCH" }

type LoopNode struct {
	ID    string
	Count int32
	Logic []ExecutableNode
}

func (n *LoopNode) NodeID() string { return n.ID }
func (n *LoopNode) Type() string   { return "LOOP" }

type PollNode struct {
	ID              string
	ConditionHookID string
	Logic           []ExecutableNode
	Interval        time.Duration
	Timeout         time.Duration
	MaxAttempts     int32
}

func (n *PollNode) NodeID() string { return n.ID }
func (n *PollNode) Type() string   { return "POLL" }

type BarrierNode struct {
	ID     string
	Name   string
	Quorum int
}

func (n *BarrierNode) NodeID() string { return n.ID }
func (n *BarrierNode) Type() string   { return "BARRIER" }

type SetNode struct {
	ID        string
	Key       string
	ValueJSON template.Expression
	Scope     string
}

func (n *SetNode) NodeID() string { return n.ID }
func (n *SetNode) Type() string   { return "SET" }

type DistributeNode struct {
	ID        string
	Key       string
	ItemsJSON template.Expression
}

func (n *DistributeNode) NodeID() string { return n.ID }
func (n *DistributeNode) Type() string   { return "DISTRIBUTE" }

type MetricNode struct {
	ID         string
	MetricType string
	Name       string
	Value      template.Expression
}

func (n *MetricNode) NodeID() string { return n.ID }
func (n *MetricNode) Type() string   { return "METRIC" }

type ReqMutateNode struct {
	ID       string
	Metadata map[string]template.Expression
	Payload  template.Expression
}

func (n *ReqMutateNode) NodeID() string { return n.ID }
func (n *ReqMutateNode) Type() string   { return "REQ_MUTATE" }

type ResAssertNode struct {
	ID                 string
	ExpectCode         int
	ExpectBodyContains string
	Extract            map[string]string
}

func (n *ResAssertNode) NodeID() string { return n.ID }
func (n *ResAssertNode) Type() string   { return "RES_ASSERT" }

type SleepNode struct {
	ID       string
	Duration time.Duration
}

func (n *SleepNode) NodeID() string { return n.ID }
func (n *SleepNode) Type() string   { return "SLEEP" }

type ScriptNode struct {
	ID     string
	HookID string
}

func (n *ScriptNode) NodeID() string { return n.ID }
func (n *ScriptNode) Type() string   { return "SCRIPT" }

type LogNode struct {
	ID      string
	Message template.Expression
}

func (n *LogNode) NodeID() string { return n.ID }
func (n *LogNode) Type() string   { return "LOG" }

type TagNode struct {
	ID    string
	Key   string
	Value template.Expression
}

func (n *TagNode) NodeID() string { return n.ID }
func (n *TagNode) Type() string   { return "TAG" }

type Graph struct {
	Setup     []ExecutableNode
	Execution []ExecutableNode
}

type GraphCompiler struct{}

func NewGraphCompiler() *GraphCompiler { return &GraphCompiler{} }

func (c *GraphCompiler) Compile(scenario *pb.Scenario) (*Graph, error) {
	graph := &Graph{}
	if scenario.Setup != nil {
		nodes, err := c.compilePipeline(scenario.Setup, 0)
		if err != nil {
			return nil, err
		}
		graph.Setup = nodes
	}
	if scenario.Execution != nil {
		nodes, err := c.compilePipeline(scenario.Execution, 0)
		if err != nil {
			return nil, err
		}
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
		case *pb.Node_Action:
			beforeNodes, err := c.compilePipeline(n.Action.Before, depth+1)
			if err != nil {
				return nil, err
			}

			afterNodes, err := c.compilePipeline(n.Action.After, depth+1)
			if err != nil {
				return nil, err
			}

			nodes = append(nodes, &ActionNode{
				ID:             step.Id,
				Protocol:       n.Action.Protocol,
				Method:         n.Action.Method,
				Target:         template.NewExpression(n.Action.Target),
				BeforePipeline: beforeNodes,
				AfterPipeline:  afterNodes,
			})

		case *pb.Node_Branch:
			truePath, err := c.compilePipeline(n.Branch.TruePath, depth+1)
			if err != nil {
				return nil, err
			}
			falsePath, err := c.compilePipeline(n.Branch.FalsePath, depth+1)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, &BranchNode{ID: step.Id, ConditionHookID: n.Branch.ConditionHookId, TruePath: truePath, FalsePath: falsePath})

		case *pb.Node_Loop:
			var count int32 = 1
			if loopCount, ok := n.Loop.Config.(*pb.LoopNode_Count); ok {
				count = loopCount.Count
			}
			logicPath, err := c.compilePipeline(n.Loop.Logic, depth+1)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, &LoopNode{ID: step.Id, Count: count, Logic: logicPath})

		case *pb.Node_Match:
			cases := make(map[string][]ExecutableNode, len(n.Match.Cases))
			for k, v := range n.Match.Cases {
				var err error
				cases[k], err = c.compilePipeline(v, depth+1)
				if err != nil {
					return nil, err
				}
			}
			defaultPath, err := c.compilePipeline(n.Match.DefaultPath, depth+1)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, &MatchNode{ID: step.Id, ConditionHookID: n.Match.ConditionHookId, Cases: cases, DefaultPath: defaultPath})

		case *pb.Node_Poll:
			intervalDur := 1 * time.Second
			timeoutDur := 30 * time.Second
			var maxAttempts int32 = 60
			if n.Poll.Policy != nil {
				if n.Poll.Policy.Interval != "" {
					intervalDur, _ = time.ParseDuration(n.Poll.Policy.Interval)
				}
				if n.Poll.Policy.Timeout != "" {
					timeoutDur, _ = time.ParseDuration(n.Poll.Policy.Timeout)
				}
				if n.Poll.Policy.MaxAttempts > 0 {
					maxAttempts = n.Poll.Policy.MaxAttempts
				}
			}
			logicPath, err := c.compilePipeline(n.Poll.Logic, depth+1)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, &PollNode{ID: step.Id, ConditionHookID: n.Poll.ConditionHookId, Logic: logicPath, Interval: intervalDur, Timeout: timeoutDur, MaxAttempts: maxAttempts})

		case *pb.Node_Script:
			nodes = append(nodes, &ScriptNode{ID: step.Id, HookID: n.Script.HookId})

		case *pb.Node_Sleep:
			dur := time.Duration(n.Sleep.DurationS * float64(time.Second))
			nodes = append(nodes, &SleepNode{ID: step.Id, Duration: dur})

		case *pb.Node_Log:
			nodes = append(nodes, &LogNode{
				ID:      step.Id,
				Message: template.NewExpression(n.Log.Message),
			})

		case *pb.Node_Tag:
			nodes = append(nodes, &TagNode{
				ID:    step.Id,
				Key:   n.Tag.Key,
				Value: template.NewExpression(n.Tag.Value),
			})

		case *pb.Node_Barrier:
			nodes = append(nodes, &BarrierNode{ID: step.Id, Name: n.Barrier.Name, Quorum: int(n.Barrier.Quorum)})

		case *pb.Node_Set:
			nodes = append(nodes, &SetNode{
				ID:        step.Id,
				Key:       n.Set.Key,
				ValueJSON: template.NewExpression(n.Set.ValueJson),
				Scope:     n.Set.Scope,
			})

		case *pb.Node_Distribute:
			nodes = append(nodes, &DistributeNode{
				ID:        step.Id,
				Key:       n.Distribute.Key,
				ItemsJSON: template.NewExpression(n.Distribute.ItemsJson),
			})

		case *pb.Node_Metric:
			nodes = append(nodes, &MetricNode{
				ID:         step.Id,
				MetricType: n.Metric.MetricType,
				Name:       n.Metric.Name,
				Value:      template.NewExpression(n.Metric.Value),
			})

		case *pb.Node_ReqMutate:
			metaMap := make(map[string]template.Expression, len(n.ReqMutate.Metadata))
			for k, v := range n.ReqMutate.Metadata {
				metaMap[k] = template.NewExpression(v)
			}
			nodes = append(nodes, &ReqMutateNode{
				ID:       step.Id,
				Metadata: metaMap,
				Payload:  template.NewExpression(n.ReqMutate.Payload),
			})

		case *pb.Node_ResAssert:
			nodes = append(nodes, &ResAssertNode{
				ID:                 step.Id,
				ExpectCode:         int(n.ResAssert.ExpectCode),
				ExpectBodyContains: n.ResAssert.ExpectPayloadContains,
				Extract:            n.ResAssert.Extract,
			})
		}
	}
	return nodes, nil
}
