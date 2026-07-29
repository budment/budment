package endpoint

import (
	"bytes"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// MarshalAST intelligently merges the compiled Endpoint into the existing YAML AST,
// ensuring that all user comments, spacings, and directives are completely preserved.
func MarshalAST(ep *Endpoint, existingData []byte) []byte {
	var doc yaml.Node
	if len(existingData) > 0 {
		if err := yaml.Unmarshal(existingData, &doc); err != nil || len(doc.Content) == 0 {
			doc = createEmptyAST()
		}
	} else {
		doc = createEmptyAST()
	}

	root := doc.Content[0]

	// Convert Endpoint into a fast lookup map for reconciliation
	expectedPaths := make(map[string]string)

	for _, ident := range ep.Identities {
		pathKey := "response." + strings.Join(ident.NodePath, ".")
		val := "identity" // Canonical Root default
		if ident.Status == StatusPending {
			val = "?"
		} else if ident.Status == StatusIgnored {
			val = "ignore"
		} else if ident.TargetID != "" {
			val = ident.TargetID // Branch Identity mapping
		}
		expectedPaths[pathKey] = val
	}

	for _, rel := range ep.Relatives {
		pathKey := strings.Join(rel.NodePath, ".")
		val := "?"
		switch rel.Status {
		case StatusResolved:
			val = rel.TargetID
		case StatusIgnored:
			val = "ignore"
		}
		expectedPaths[pathKey] = val
	}

	// Reconcile AST: Upsert missing/changed, Delete obsolete
	reconcileAST(root, "", expectedPaths)
	sortAST(root)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	_ = encoder.Encode(&doc)
	encoder.Close()

	return bytes.TrimSpace(buf.Bytes())
}

func createEmptyAST() yaml.Node {
	return yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{{Kind: yaml.MappingNode}},
	}
}

// reconcileAST walks the YAML tree, adds missing expected keys, updates values, and removes obsolete nodes.
func reconcileAST(node *yaml.Node, currentPath string, expectedPaths map[string]string) {
	if node.Kind != yaml.MappingNode {
		return
	}

	existingLocalKeys := make(map[string]bool)
	var newContents []*yaml.Node

	// Retain and update valid existing nodes
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		localKey := keyNode.Value

		fullPath := localKey
		if currentPath != "" {
			fullPath = currentPath + "." + localKey
		}
		existingLocalKeys[localKey] = true

		keep := false
		for expPath := range expectedPaths {
			if expPath == fullPath || strings.HasPrefix(expPath, fullPath+".") {
				keep = true
				break
			}
		}

		if keep {
			switch valNode.Kind {
			case yaml.ScalarNode:
				if expectedVal, exists := expectedPaths[fullPath]; exists {
					valNode.Value = expectedVal // Safe Value mutation (preserves inline comments)
				}
			case yaml.MappingNode:
				reconcileAST(valNode, fullPath, expectedPaths)
			}
			newContents = append(newContents, keyNode, valNode)
		}
	}

	node.Content = newContents

	// Find and generate missing nodes
	missingLocalKeys := make(map[string]bool)
	for expPath := range expectedPaths {
		if currentPath == "" || strings.HasPrefix(expPath, currentPath+".") {
			relPath := expPath
			if currentPath != "" {
				relPath = strings.TrimPrefix(expPath, currentPath+".")
			}
			localKey := strings.SplitN(relPath, ".", 2)[0]
			if !existingLocalKeys[localKey] {
				missingLocalKeys[localKey] = true
			}
		}
	}

	// Deterministic Missing Key Sorting
	var sortedMissing []string
	for k := range missingLocalKeys {
		sortedMissing = append(sortedMissing, k)
	}
	sort.Strings(sortedMissing)

	for _, localKey := range sortedMissing {
		fullPath := localKey
		if currentPath != "" {
			fullPath = currentPath + "." + localKey
		}

		if expectedVal, isLeaf := expectedPaths[fullPath]; isLeaf {
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: localKey},
				&yaml.Node{Kind: yaml.ScalarNode, Value: expectedVal},
			)
		} else {
			branchNode := &yaml.Node{Kind: yaml.MappingNode}
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: localKey},
				branchNode,
			)
			reconcileAST(branchNode, fullPath, expectedPaths) // Deep recursion
		}
	}
}

func sortAST(node *yaml.Node) {
	if node.Kind != yaml.MappingNode {
		return
	}

	type pair struct {
		k, v   *yaml.Node
		weight int
	}

	var pairs []pair
	for i := 0; i < len(node.Content); i += 2 {
		pairs = append(pairs, pair{
			k:      node.Content[i],
			v:      node.Content[i+1],
			weight: sortWeight(node.Content[i].Value),
		})
	}

	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].weight != pairs[j].weight {
			return pairs[i].weight < pairs[j].weight
		}
		return pairs[i].k.Value < pairs[j].k.Value
	})

	node.Content = nil
	for _, p := range pairs {
		node.Content = append(node.Content, p.k, p.v)
		sortAST(p.v)
	}
}

func sortWeight(key string) int {
	switch key {
	case "response":
		return 1
	case "path":
		return 2
	case "query":
		return 3
	case "header":
		return 4
	case "cookie":
		return 5
	case "body":
		return 6
	default:
		return 100
	}
}
