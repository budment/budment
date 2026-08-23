package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/vunas/blaster/internal/planner"
	"github.com/vunas/blaster/internal/tui/theme"
)

func PrintPhase(phaseName string, nodes []planner.ExecutableNode) {
	fmt.Printf("\n%s\n", theme.TextMagenta("▶ "+strings.ToUpper(phaseName)+" PHASE"))
	fmt.Println(theme.TextDim(strings.Repeat("═", 60)))

	if len(nodes) == 0 {
		fmt.Printf("  %s\n", theme.TextDim("(empty)"))
		return
	}

	printTreeReal(nodes, "")
}

func PrintLifecycleTree(nodes []planner.ExecutableNode, depth int) {
	printTreeReal(nodes, strings.Repeat("  ", depth))
}

func printTreeReal(nodes []planner.ExecutableNode, prefix string) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		connector := "├──"
		childPrefix := prefix + "│   "

		if isLast {
			connector = "└──"
			childPrefix = prefix + "    "
		}

		nodeID := theme.TextDim(fmt.Sprintf("[id: %s]", node.NodeID()))

		switch n := node.(type) {
		case *planner.HttpNode:
			printHTTPNode(prefix, connector, childPrefix, n, nodeID)
		case *planner.BranchNode:
			printBranchNode(prefix, connector, childPrefix, n, nodeID)
		case *planner.LoopNode:
			printLoopNode(prefix, connector, childPrefix, n, nodeID)
		case *planner.MatchNode:
			printMatchNode(prefix, connector, childPrefix, n, nodeID)
		case *planner.PollNode:
			printPollNode(prefix, connector, childPrefix, n, nodeID)
		case *planner.ScriptNode:
			fmt.Printf("%s%s %s %s %s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextMagenta("SCRIPT"), n.HookID, nodeID)
		}
	}
}

func printHTTPNode(prefix, connector, childPrefix string, n *planner.HttpNode, nodeID string) {
	method := theme.TextGreen(n.Method)
	if n.Method == "PUT" || n.Method == "DELETE" {
		method = theme.TextYellow(n.Method)
	}

	fmt.Printf("%s%s %s %s %s %s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextCyan("[HTTP]"), method, n.URL, nodeID)

	type stepItem struct {
		isGroup bool
		label   string
		items   []string
	}

	var steps []stepItem

	if len(n.Injects) > 0 {
		list := make([]string, len(n.Injects))
		for i, inj := range n.Injects {
			list[i] = fmt.Sprintf("%s → %s", theme.TextYellow(inj.StackID), inj.Target)
		}
		steps = append(steps, stepItem{isGroup: true, label: theme.TextBlue("[INJECT]"), items: list})
	}

	if n.BeforeHookID != "" {
		steps = append(steps, stepItem{isGroup: false, label: fmt.Sprintf("%s %s", theme.TextMagenta("[BEFORE]"), n.BeforeHookID)})
	}

	if n.AfterHookID != "" {
		steps = append(steps, stepItem{isGroup: false, label: fmt.Sprintf("%s %s", theme.TextMagenta("[AFTER]"), n.AfterHookID)})
	}

	if len(n.Extracts) > 0 {
		list := make([]string, len(n.Extracts))
		for i, ext := range n.Extracts {
			list[i] = fmt.Sprintf("%s → %s", ext.Path, theme.TextYellow(ext.StackID))
		}
		steps = append(steps, stepItem{isGroup: true, label: theme.TextBlue("[EXTRACT]"), items: list})
	}

	for i, step := range steps {
		isLastStep := i == len(steps)-1
		stepConn := "├──"
		subPrefix := childPrefix + "│   "
		if isLastStep {
			stepConn = "└──"
			subPrefix = childPrefix + "    "
		}

		fmt.Printf("%s%s %s\n", theme.TextDim(childPrefix), theme.TextDim(stepConn), step.label)

		if step.isGroup {
			for j, item := range step.items {
				itemConn := "├──"
				if j == len(step.items)-1 {
					itemConn = "└──"
				}
				fmt.Printf("%s%s %s\n", theme.TextDim(subPrefix), theme.TextDim(itemConn), item)
			}
		}
	}
}

func printBranchNode(prefix, connector, childPrefix string, n *planner.BranchNode, nodeID string) {
	fmt.Printf("%s%s %s Condition: %s %s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextYellow("[BRANCH]"), n.ConditionHookID, nodeID)
	fmt.Printf("%s├── %s\n", theme.TextDim(childPrefix), theme.TextGreen("[TRUE]"))
	printTreeReal(n.TruePath, childPrefix+"│   ")

	fmt.Printf("%s└── %s\n", theme.TextDim(childPrefix), theme.TextRed("[FALSE]"))
	if len(n.FalsePath) > 0 {
		printTreeReal(n.FalsePath, childPrefix+"    ")
	} else {
		fmt.Printf("%s    %s\n", theme.TextDim(childPrefix), theme.TextDim("(empty)"))
	}
}

func printLoopNode(prefix, connector, childPrefix string, n *planner.LoopNode, nodeID string) {
	fmt.Printf("%s%s %s %d iterations %s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextBlue("[LOOP]"), n.Count, nodeID)
	printTreeReal(n.Logic, childPrefix)
}

func printMatchNode(prefix, connector, childPrefix string, n *planner.MatchNode, nodeID string) {
	fmt.Printf("%s%s %s Hook: %s %s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextMagenta("[MATCH]"), n.ConditionHookID, nodeID)

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

		fmt.Printf("%s%s %s\n", theme.TextDim(childPrefix), theme.TextDim(caseConnector), theme.TextCyan("Case: "+caseName))
		printTreeReal(path, caseChildPrefix)
	}

	if len(n.DefaultPath) > 0 {
		fmt.Printf("%s└── %s\n", theme.TextDim(childPrefix), theme.TextYellow("Default"))
		printTreeReal(n.DefaultPath, childPrefix+"    ")
	}
}

func printPollNode(prefix, connector, childPrefix string, n *planner.PollNode, nodeID string) {
	fmt.Printf("%s%s %s Interval: %s, Max: %d %s\n", theme.TextDim(prefix), theme.TextDim(connector), theme.TextCyan("[POLL]"), n.Interval, n.MaxAttempts, nodeID)
	fmt.Printf("%s├── %s Condition: %s\n", theme.TextDim(childPrefix), theme.TextYellow("[CHECK]"), n.ConditionHookID)
	fmt.Printf("%s└── %s\n", theme.TextDim(childPrefix), theme.TextGreen("[LOGIC]"))
	printTreeReal(n.Logic, childPrefix+"    ")
}
