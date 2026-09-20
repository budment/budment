package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

type stubScope struct {
	vars map[string]any
}

func (s *stubScope) Resolve(name string) (any, bool) {
	v, ok := s.vars[name]
	return v, ok
}

func TestTemplate_StaticExpression(t *testing.T) {
	expr := NewExpression("https://api.example.com/health")

	if !expr.IsStatic() {
		t.Fatal("expected expression without template braces to be static")
	}

	res := expr.Render(nil)
	if res != "https://api.example.com/health" {
		t.Fatalf("rendered string mismatch: expected 'https://api.example.com/health', got '%s'", res)
	}
}

func TestTemplate_DynamicExpressions(t *testing.T) {
	// Configure test environment variables
	t.Setenv("BUDMENT_REGION", "ap-southeast-1")

	scope := &stubScope{
		vars: map[string]any{
			"user_id": 1001,
			"token":   "bearer_secret_xyz",
		},
	}

	rawTmpl := "https://api.com/users/{{user_id}}?region={{@env:BUDMENT_REGION:us-east-1}}&env={{@env:NOT_SET:fallback_val}}&token={{token}}"
	expr := NewExpression(rawTmpl)

	if expr.IsStatic() {
		t.Fatal("expected expression to be dynamic")
	}

	rendered := expr.Render(scope)
	expected := "https://api.com/users/1001?region=ap-southeast-1&env=fallback_val&token=bearer_secret_xyz"

	if rendered != expected {
		t.Fatalf("rendered template mismatch:\nexpected: %s\ngot:      %s", expected, rendered)
	}
}

func TestTemplate_RandomGenerators(t *testing.T) {
	// 1. UUID v4
	uuid := FastUUID()
	if len(uuid) != 36 || uuid[14] != '4' {
		t.Fatalf("invalid UUID v4 format: %s", uuid)
	}

	// 2. Random string
	str16 := FastRandomString(16)
	if len(str16) != 16 {
		t.Fatalf("random string length mismatch: expected 16, got %d", len(str16))
	}

	// 3. Random integer
	val := FastRandomInt(10, 20)
	if val < 10 || val > 20 {
		t.Fatalf("FastRandomInt out of bounds [10, 20]: %d", val)
	}

	// 4. Random pick
	items := []any{"apple", "banana", "cherry"}
	picked := FastRandomPick(items)
	if picked != "apple" && picked != "banana" && picked != "cherry" {
		t.Fatalf("FastRandomPick picked invalid element: %v", picked)
	}

	// 5. Template AST pick and random integer
	expr := NewExpression("Role: {{@random:pick:admin,editor,guest}} | Age: {{@random:int:20:30}}")
	res := expr.Render(nil)

	if !strings.HasPrefix(res, "Role: ") || !strings.Contains(res, " | Age: ") {
		t.Fatalf("template rendering with pick/int failed: %s", res)
	}
}

func TestTemplate_FileCaching_ZeroDiskReadOnSubsequentCalls(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "payload.json")
	initialContent := `{"message":"cached"}`

	if err := os.WriteFile(filePath, []byte(initialContent), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Call 1: Read from disk and populate cache
	content1 := GetFileContent(filePath)
	if content1 != initialContent {
		t.Fatalf("initial content mismatch: expected %s, got %s", initialContent, content1)
	}

	// Remove physical file from disk to verify subsequent reads resolve from RAM cache
	if err := os.Remove(filePath); err != nil {
		t.Fatalf("failed to remove temp file: %v", err)
	}

	// Call 2: Read directly from sync.Map cache
	content2 := GetFileContent(filePath)
	if content2 != initialContent {
		t.Fatalf("cached content mismatch: expected %s, got %s", initialContent, content2)
	}
}

func BenchmarkTemplate_Render_Dynamic(b *testing.B) {
	scope := &stubScope{
		vars: map[string]any{"id": 42},
	}
	expr := NewExpression("https://api.internal/v1/items/{{id}}?uuid={{@random:uuid}}")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = expr.Render(scope)
	}
}

func TestTemplate_DeepLookup_And_Fallback(t *testing.T) {
	// Mock a Goja Runtime object to simulate JS hook state injection
	vm := goja.New()
	jsObjVal, _ := vm.RunString(`({ profile: { role: "super_admin", tags: ["a", "b"] } })`)
	jsObj := jsObjVal.ToObject(vm)

	scope := &stubScope{
		vars: map[string]any{
			"user": map[string]any{
				"items": []any{
					map[string]any{"id": 42},
				},
			},
			"js_data": jsObj,
		},
	}

	// 1. Verify traversal through Go native maps and slices using '}{' delimiters
	expr1 := NewExpression("Item ID: {{user}{items}{0}{id}}")
	if res := expr1.Render(scope); res != "Item ID: 42" {
		t.Errorf("native map/slice lookup failed: expected 'Item ID: 42', got '%s'", res)
	}

	// 2. Verify property access
	expr2 := NewExpression("Role: {{js_data}{profile}{role}}")
	if res := expr2.Render(scope); res != "Role: super_admin" {
		t.Errorf("goja object lookup failed: expected 'Role: super_admin', got '%s'", res)
	}

	// 3. Verify graceful fallback to raw template string on broken paths
	expr3 := NewExpression("Bad Path: {{user}{wrong}{path}}")
	if res := expr3.Render(scope); res != "Bad Path: {{user}{wrong}{path}}" {
		t.Errorf("broken path fallback failed: expected raw string, got '%s'", res)
	}

	// 4. Verify JSON serialization dump for zero-path root object resolution
	expr4 := NewExpression("Dump: {{user}}")
	res4 := expr4.Render(scope)
	if !strings.Contains(res4, `"items"`) {
		t.Errorf("root object JSON dump failed: expected JSON payload, got '%s'", res4)
	}
}
