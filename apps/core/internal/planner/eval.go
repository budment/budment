package planner

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	pb "github.com/vunas/blaster/internal/planner/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

func (e *Evaluator) Evaluate(jsBundle string) (*pb.Scenario, error) {
	vm := goja.New()

	// Evaluate the compiled JS bundle.
	_, err := vm.RunString(jsBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate JS bundle: %w", err)
	}

	// Resolve the exported AST from the bundle.
	astValue, err := vm.RunString("__BLASTER_AST__.default")
	if err != nil || astValue == nil || goja.IsUndefined(astValue) {

		// Fall back to the raw AST when no default export is present.
		astValue, err = vm.RunString("__BLASTER_AST__")
		if err != nil || astValue == nil || goja.IsUndefined(astValue) {
			return nil, fmt.Errorf("không thể tìm thấy AST trong Goja VM. Đảm bảo file JS có export kịch bản. Lỗi: %v", err)
		}
	}

	// Serialize the JS AST to JSON.
	astObj := astValue.ToObject(vm)
	rawJSON, err := json.Marshal(astObj.Export())
	if err != nil {
		return nil, fmt.Errorf("failed to serialize JS AST into JSON: %w", err)
	}

	// Deserialize the JSON into the Protobuf Scenario.
	var scenario pb.Scenario
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true, // Ignore unknown fields for forward compatibility.
	}

	if err := unmarshaler.Unmarshal(rawJSON, &scenario); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into Protobuf Scenario: %w\nJSON sinh ra là: %s", err, string(rawJSON))
	}

	return &scenario, nil
}
