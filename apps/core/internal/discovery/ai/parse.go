package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/discovery/endpoint"
)

type Resolver struct {
	cfg config.AIConfig
	log *slog.Logger
}

func NewResolver(cfg config.AIConfig, log *slog.Logger) *Resolver {
	return &Resolver{
		cfg: cfg,
		log: log,
	}
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []message       `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Temperature    float32         `json:"temperature"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type aiDecision struct {
	Mappings []struct {
		PendingName string `json:"pending_name"`
		EndpointKey string `json:"endpoint_key"`
		NodePath    string `json:"node_path"`
		TargetID    string `json:"target_id"`
	} `json:"mappings"`
}

// pendingItem is a temporary struct to hold memory pointers for in-place mutation
type pendingItem struct {
	EndpointKey string
	Name        string
	NodePath    string
	SetTarget   func(target string) // Closure to mutate the actual Endpoint pointer
}

// Resolve scans endpoints, builds the prompt, calls the LLM, and mutates in-place.
func (r *Resolver) Resolve(endpoints []*endpoint.Endpoint) error {
	var available []string
	var pendingList []pendingItem
	pendingMap := make(map[string]pendingItem)

	// Build Global Index & Pending Basket directly from Endpoints
	for _, ep := range endpoints {
		epKey := ep.Method + ":" + ep.Path

		for i := range ep.Identities {
			ident := &ep.Identities[i]
			if ident.Status == endpoint.StatusResolved && ident.TargetID == "" {
				targetID := ep.Method + ":" + ep.Path + ":" + ident.Name
				typeStr := strings.Join(ident.Types, ", ")
				available = append(available, "- "+targetID+" (Types: ["+typeStr+"])")
			} else if ident.Status == endpoint.StatusPending {
				nodePathStr := strings.Join(ident.NodePath, ".")
				item := pendingItem{
					EndpointKey: epKey,
					Name:        ident.Name,
					NodePath:    nodePathStr,
					SetTarget: func(target string) {
						ident.Status = endpoint.StatusResolved
						ident.TargetID = target
					},
				}
				pendingList = append(pendingList, item)

				mapKey := epKey + "|" + ident.Name + "|" + nodePathStr
				pendingMap[mapKey] = item
			}
		}

		for i := range ep.Relatives {
			rel := &ep.Relatives[i]
			if rel.Status == endpoint.StatusPending {
				nodePathStr := strings.Join(rel.NodePath, ".")
				item := pendingItem{
					EndpointKey: epKey,
					Name:        rel.Name,
					NodePath:    nodePathStr,
					SetTarget: func(target string) {
						rel.Status = endpoint.StatusResolved
						rel.TargetID = target
					},
				}
				pendingList = append(pendingList, item)

				mapKey := epKey + "|" + rel.Name + "|" + nodePathStr
				pendingMap[mapKey] = item
			}
		}
	}

	if len(pendingList) == 0 {
		return nil
	}

	systemPrompt := `You are an expert API mapping assistant. 
Your task is to connect unresolved request/response fields to the best matching Root Identity.
RULES:
1. ONLY return a strictly valid JSON object matching this schema: {"mappings": [{"pending_name": "string", "endpoint_key": "string", "node_path": "string", "target_id": "string"}]}
2. DO NOT wrap the JSON in markdown code blocks.
3. If you cannot confidently map a field based on standard REST semantics, omit it from the mappings array.
4. Ensure "node_path" exactly matches the provided input.`

	userPrompt := r.buildUserPrompt(available, pendingList)

	reqPayload := chatRequest{
		Model: r.cfg.Model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		Temperature:    0.1, // Keep it deterministic
	}

	reqBytes, _ := json.Marshal(reqPayload)

	// Call HTTP LLM API
	endpointURL := strings.TrimRight(r.cfg.BaseURL, "/") + "/chat/completions"
	client := &http.Client{Timeout: 30 * time.Second}

	var resp *http.Response
	var err error
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, _ := http.NewRequest("POST", endpointURL, bytes.NewReader(reqBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)

		resp, err = client.Do(req)

		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			bodyErr, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("ai client error (status %d): %s", resp.StatusCode, string(bodyErr))
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < maxRetries {
			sleepDur := time.Duration(1<<attempt) * time.Second
			r.log.Warn("AI Network issue or Rate limit, retrying...", "attempt", attempt+1, "sleep", sleepDur, "error", err)
			time.Sleep(sleepDur)
		}
	}

	if err != nil {
		return fmt.Errorf("ai network error after retries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ai failed with status %d after retries", resp.StatusCode)
	}

	// Parse AI Response
	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return fmt.Errorf("failed to decode ai response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return fmt.Errorf("ai returned empty choices")
	}

	rawContent := chatResp.Choices[0].Message.Content
	var decision aiDecision
	if err := json.Unmarshal([]byte(rawContent), &decision); err != nil {
		return fmt.Errorf("failed to parse AI JSON decision: %w\nRaw: %s", err, rawContent)
	}

	// Apply Results back to memory pointers
	resolvedCount := 0
	for _, mapping := range decision.Mappings {
		mapKey := mapping.EndpointKey + "|" + mapping.PendingName + "|" + mapping.NodePath

		if p, exists := pendingMap[mapKey]; exists {
			p.SetTarget(mapping.TargetID)
			resolvedCount++
			r.log.Debug("AI Resolved Mapping", "endpoint", p.EndpointKey, "field", p.Name, "target", mapping.TargetID)
		} else {
			r.log.Warn("AI hallucinated or returned mismatched mapping", "lookup_key", mapKey)
		}
	}

	r.log.Info("AI Resolution phase completed", "resolved_by_ai", resolvedCount, "remaining", len(pendingList)-resolvedCount)
	return nil
}

func (r *Resolver) buildUserPrompt(available []string, pending []pendingItem) string {
	var builder strings.Builder

	builder.WriteString("AVAILABLE ROOT IDENTITIES (Global Index):\n")
	for _, a := range available {
		builder.WriteString(a)
		builder.WriteString("\n")
	}

	builder.WriteString("\nUNRESOLVED FIELDS (Pending Basket):\n")
	for _, p := range pending {
		builder.WriteString(fmt.Sprintf("- Name: %s (Endpoint: %s, NodePath: %s)\n", p.Name, p.EndpointKey, p.NodePath))
	}

	builder.WriteString("\nPlease map the UNRESOLVED FIELDS to the most logically related AVAILABLE ROOT IDENTITIES. Ensure schema types are compatible.")
	return builder.String()
}
