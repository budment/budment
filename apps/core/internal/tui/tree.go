package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/vunas/blaster/internal/planner"
	"github.com/vunas/blaster/internal/tui/theme"
)

func getVisibleNodes(nodes []planner.ExecutableNode, detailed bool) []planner.ExecutableNode {
	if detailed {
		return nodes
	}
	var visible []planner.ExecutableNode
	for _, n := range nodes {
		switch n.(type) {
		case *planner.ActionNode, *planner.BranchNode, *planner.LoopNode, *planner.MatchNode, *planner.PollNode:
			visible = append(visible, n)
		}
	}
	return visible
}

func PrintPhase(phaseName string, nodes []planner.ExecutableNode, detailed bool) {
	fmt.Printf("\n%s\n", theme.TextMagenta("▶ "+strings.ToUpper(phaseName)+" PHASE"))
	PrintDivider()

	visible := getVisibleNodes(nodes, detailed)
	if len(visible) == 0 {
		fmt.Printf("  %s\n", theme.TextDim("(empty)"))
		return
	}

	printTreeReal(visible, "", detailed)
}

func PrintLifecycleTree(nodes []planner.ExecutableNode, depth int, detailed bool) {
	visible := getVisibleNodes(nodes, detailed)
	printTreeReal(visible, strings.Repeat("  ", depth), detailed)
}

func printTreeReal(nodes []planner.ExecutableNode, prefix string, detailed bool) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		connector := "├──"
		childPrefix := prefix + "│   "

		if isLast {
			connector = "└──"
			childPrefix = prefix + "    "
		}

		nodeID := " " + theme.TextDim(fmt.Sprintf("[id: %s]", node.NodeID()))

		switch n := node.(type) {
		case *planner.ActionNode:
			printHTTPNode(prefix, connector, childPrefix, n, nodeID, detailed)
		case *planner.BranchNode:
			printBranchNode(prefix, connector, childPrefix, n, nodeID, detailed)
		case *planner.LoopNode:
			printLoopNode(prefix, connector, childPrefix, n, nodeID, detailed)
		case *planner.MatchNode:
			printMatchNode(prefix, connector, childPrefix, n, nodeID, detailed)
		case *planner.PollNode:
			printPollNode(prefix, connector, childPrefix, n, nodeID, detailed)
		case *planner.ScriptNode:
			fmt.Printf("%s%s %s %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextMagenta("[SCRIPT]"), n.HookID, nodeID)
		case *planner.SleepNode:
			fmt.Printf("%s%s %s %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextBlue("[SLEEP]"), n.Duration.String(), nodeID)
		case *planner.LogNode:
			fmt.Printf("%s%s %s \"%s\"%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextCyan("[LOG]"), n.Message.Raw, nodeID)
		case *planner.BarrierNode:
			fmt.Printf("%s%s %s Quorum: %d%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextRed("[BARRIER]"), n.Quorum, nodeID)
		case *planner.SetNode:
			fmt.Printf("%s%s %s %s = %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextCyan("[SET]"), n.Key, n.ValueJSON.Raw, nodeID)
		case *planner.DistributeNode:
			fmt.Printf("%s%s %s %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextYellow("[DISTRIBUTE]"), n.Key, nodeID)
		case *planner.MetricNode:
			fmt.Printf("%s%s %s %s: %s = %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextMagenta("[METRIC]"), n.MetricType, n.Name, n.Value.Raw, nodeID)
		case *planner.ReqMutateNode:
			payloadTag := ""
			if n.Payload.Raw != "" {
				payloadTag = " (+Payload)"
			}
			fmt.Printf("%s%s %s Headers: %d%s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextBlue("[REQ_MUTATE]"), len(n.Metadata), payloadTag, nodeID)
		case *planner.ResAssertNode:
			extractTag := ""
			if len(n.Extract) > 0 {
				extractTag = fmt.Sprintf(" | Extract: %d", len(n.Extract))
			}
			fmt.Printf("%s%s %s Expect Status: %d%s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextMagenta("[RES_ASSERT]"), n.ExpectCode, extractTag, nodeID)
		}
	}
}

func summarizePipeline(nodes []planner.ExecutableNode) string {
	if len(nodes) == 0 {
		return ""
	}
	var parts []string
	for _, n := range nodes {
		switch n.(type) {
		case *planner.ScriptNode:
			parts = append(parts, "script")
		case *planner.SetNode:
			parts = append(parts, "set")
		case *planner.MetricNode:
			parts = append(parts, "metric")
		case *planner.LogNode:
			parts = append(parts, "log")
		case *planner.SleepNode:
			parts = append(parts, "sleep")
		case *planner.DistributeNode:
			parts = append(parts, "distribute")
		case *planner.ReqMutateNode:
			parts = append(parts, "req_mutate")
		case *planner.ResAssertNode:
			parts = append(parts, "res_assert")
		case *planner.BarrierNode:
			parts = append(parts, "barrier")
		default:
			parts = append(parts, strings.ToLower(n.Type()))
		}
	}
	return strings.Join(parts, ", ")
}

func formatNodeInline(node planner.ExecutableNode) string {
	switch n := node.(type) {
	case *planner.ScriptNode:
		return theme.TextMagenta("[SCRIPT] ") + n.HookID
	case *planner.SetNode:
		return theme.TextCyan("[SET] ") + n.Key + " = " + n.ValueJSON.Raw
	case *planner.MetricNode:
		return theme.TextMagenta("[METRIC] ") + n.MetricType + ": " + n.Name
	case *planner.LogNode:
		return theme.TextCyan("[LOG] ") + n.Message.Raw
	case *planner.SleepNode:
		return theme.TextBlue("[SLEEP] ") + n.Duration.String()
	case *planner.DistributeNode:
		return theme.TextYellow("[DISTRIBUTE] ") + n.Key
	case *planner.ReqMutateNode:
		return theme.TextBlue("[REQ_MUTATE]")
	case *planner.ResAssertNode:
		return theme.TextMagenta("[RES_ASSERT] ") + fmt.Sprintf("Expect Status: %d", n.ExpectCode)
	default:
		return fmt.Sprintf("[%s]", node.Type())
	}
}

func printHTTPNode(prefix, connector, childPrefix string, n *planner.ActionNode, nodeID string, detailed bool) {
	method := theme.TextGreen(n.Method)
	if n.Method == "PUT" || n.Method == "DELETE" || n.Method == "PATCH" {
		method = theme.TextYellow(n.Method)
	}

	fmt.Printf("%s%s %s %s %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextCyan("[HTTP]"), method, n.Target.Raw, nodeID)

	hasBefore := len(n.BeforePipeline) > 0
	hasAfter := len(n.AfterPipeline) > 0

	if !detailed {
		var blocks []string

		if hasBefore {
			blocks = append(blocks, "before")
		}
		if hasAfter {
			blocks = append(blocks, "after")
		}

		for i, b := range blocks {
			isLast := i == len(blocks)-1
			conn := "├──"
			if isLast {
				conn = "└──"
			}
			switch b {
			case "before":
				fmt.Printf("%s%s %s → %s\n", theme.TextDim(childPrefix), theme.TextDim(conn), theme.TextMagenta("[BEFORE]"), theme.TextDim(summarizePipeline(n.BeforePipeline)))
			case "after":
				fmt.Printf("%s%s %s → %s\n", theme.TextDim(childPrefix), theme.TextDim(conn), theme.TextMagenta("[AFTER]"), theme.TextDim(summarizePipeline(n.AfterPipeline)))
			}
		}
		return
	}

	printGroupInline := func(label string, content string) {
		conn := "├──"
		fmt.Printf("%s%s %s → %s\n", theme.TextDim(childPrefix), theme.TextDim(conn), label, content)
	}

	if hasBefore {
		isLast := !hasAfter
		if len(n.BeforePipeline) == 1 {
			printGroupInline(theme.TextMagenta("[BEFORE]"), formatNodeInline(n.BeforePipeline[0]))
		} else {
			conn := "├──"
			subPref := childPrefix + "│   "
			if isLast {
				conn = "└──"
				subPref = childPrefix + "    "
			}
			fmt.Printf("%s%s %s\n", theme.TextDim(childPrefix), theme.TextDim(conn), theme.TextMagenta("[BEFORE]"))
			printTreeReal(n.BeforePipeline, subPref, detailed)
		}
	}

	if hasAfter {
		if len(n.AfterPipeline) == 1 {
			printGroupInline(theme.TextMagenta("[AFTER]"), formatNodeInline(n.AfterPipeline[0]))
		} else {
			conn := "├──"
			subPref := childPrefix + "│   "
			fmt.Printf("%s%s %s\n", theme.TextDim(childPrefix), theme.TextDim(conn), theme.TextMagenta("[AFTER]"))
			printTreeReal(n.AfterPipeline, subPref, detailed)
		}
	}
}

func printBranchNode(prefix, connector, childPrefix string, n *planner.BranchNode, nodeID string, detailed bool) {
	fmt.Printf("%s%s %s Condition: %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextYellow("[BRANCH]"), n.ConditionHookID, nodeID)

	trueVisible := getVisibleNodes(n.TruePath, detailed)
	if len(trueVisible) > 0 {
		fmt.Printf("%s├── %s\n", theme.TextDim(childPrefix), theme.TextGreen("[TRUE]"))
		printTreeReal(trueVisible, childPrefix+"│   ", detailed)
	} else if len(n.TruePath) > 0 {
		fmt.Printf("%s├── %s → %s\n", theme.TextDim(childPrefix), theme.TextGreen("[TRUE]"), theme.TextDim(summarizePipeline(n.TruePath)))
	} else {
		fmt.Printf("%s├── %s %s\n", theme.TextDim(childPrefix), theme.TextGreen("[TRUE]"), theme.TextDim("(empty)"))
	}

	falseVisible := getVisibleNodes(n.FalsePath, detailed)
	if len(falseVisible) > 0 {
		fmt.Printf("%s└── %s\n", theme.TextDim(childPrefix), theme.TextRed("[FALSE]"))
		printTreeReal(falseVisible, childPrefix+"    ", detailed)
	} else if len(n.FalsePath) > 0 {
		fmt.Printf("%s└── %s → %s\n", theme.TextDim(childPrefix), theme.TextRed("[FALSE]"), theme.TextDim(summarizePipeline(n.FalsePath)))
	} else {
		fmt.Printf("%s└── %s %s\n", theme.TextDim(childPrefix), theme.TextRed("[FALSE]"), theme.TextDim("(empty)"))
	}
}

func printLoopNode(prefix, connector, childPrefix string, n *planner.LoopNode, nodeID string, detailed bool) {
	visible := getVisibleNodes(n.Logic, detailed)
	if len(visible) > 0 {
		fmt.Printf("%s%s %s %d iterations%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextBlue("[LOOP]"), n.Count, nodeID)
		printTreeReal(visible, childPrefix, detailed)
	} else if len(n.Logic) > 0 {
		fmt.Printf("%s%s %s %d iterations → %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextBlue("[LOOP]"), n.Count, theme.TextDim(summarizePipeline(n.Logic)), nodeID)
	} else {
		fmt.Printf("%s%s %s %d iterations %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextBlue("[LOOP]"), n.Count, theme.TextDim("(empty)"), nodeID)
	}
}

func printMatchNode(prefix, connector, childPrefix string, n *planner.MatchNode, nodeID string, detailed bool) {
	fmt.Printf("%s%s %s Hook: %s%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextMagenta("[MATCH]"), n.ConditionHookID, nodeID)

	caseNames := make([]string, 0, len(n.Cases))
	for k := range n.Cases {
		caseNames = append(caseNames, k)
	}
	slices.Sort(caseNames)

	for i, caseName := range caseNames {
		path := n.Cases[caseName]
		isLastCase := (i == len(caseNames)-1) && len(n.DefaultPath) == 0

		caseConnector := "├──"
		caseChildPrefix := childPrefix + "│   "
		if isLastCase {
			caseConnector = "└──"
			caseChildPrefix = childPrefix + "    "
		}

		visible := getVisibleNodes(path, detailed)
		if len(visible) > 0 {
			fmt.Printf("%s%s %s\n", theme.TextDim(childPrefix), theme.TextDim(caseConnector), theme.TextCyan("Case: "+caseName))
			printTreeReal(visible, caseChildPrefix, detailed)
		} else if len(path) > 0 {
			fmt.Printf("%s%s %s → %s\n", theme.TextDim(childPrefix), theme.TextDim(caseConnector), theme.TextCyan("Case: "+caseName), theme.TextDim(summarizePipeline(path)))
		} else {
			fmt.Printf("%s%s %s %s\n", theme.TextDim(childPrefix), theme.TextDim(caseConnector), theme.TextCyan("Case: "+caseName), theme.TextDim("(empty)"))
		}
	}

	if len(n.DefaultPath) > 0 {
		visible := getVisibleNodes(n.DefaultPath, detailed)
		if len(visible) > 0 {
			fmt.Printf("%s└── %s\n", theme.TextDim(childPrefix), theme.TextYellow("Default"))
			printTreeReal(visible, childPrefix+"    ", detailed)
		} else {
			fmt.Printf("%s└── %s → %s\n", theme.TextDim(childPrefix), theme.TextYellow("Default"), theme.TextDim(summarizePipeline(n.DefaultPath)))
		}
	}
}

func printPollNode(prefix, connector, childPrefix string, n *planner.PollNode, nodeID string, detailed bool) {
	fmt.Printf("%s%s %s Interval: %s, Max: %d%s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextCyan("[POLL]"), n.Interval, n.MaxAttempts, nodeID)
	fmt.Printf("%s├── %s Condition: %s\n", theme.TextDim(childPrefix), theme.TextYellow("[CHECK]"), n.ConditionHookID)

	visible := getVisibleNodes(n.Logic, detailed)
	if len(visible) > 0 {
		fmt.Printf("%s└── %s\n", theme.TextDim(childPrefix), theme.TextGreen("[LOGIC]"))
		printTreeReal(visible, childPrefix+"    ", detailed)
	} else if len(n.Logic) > 0 {
		fmt.Printf("%s└── %s → %s\n", theme.TextDim(childPrefix), theme.TextGreen("[LOGIC]"), theme.TextDim(summarizePipeline(n.Logic)))
	} else {
		fmt.Printf("%s└── %s %s\n", theme.TextDim(childPrefix), theme.TextGreen("[LOGIC]"), theme.TextDim("(empty)"))
	}
}
