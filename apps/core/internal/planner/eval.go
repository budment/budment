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

func (e *Evaluator) Evaluate(jsBundle string) ([]*pb.Scenario, error) {
	vm := goja.New()

	// Evaluate the compiled JS bundle.
	_, err := vm.RunString(jsBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate JS bundle: %w", err)
	}

	astArrayValue, err := vm.RunString("__BLASTER_SCENARIOS__")
	if err != nil || astArrayValue == nil || goja.IsUndefined(astArrayValue) {
		return nil, fmt.Errorf("Failed to get scenarios array. Ensure that you have called .build() at least once in your script: %w", err)
	}

	astObj := astArrayValue.ToObject(vm)
	rawJSON, err := json.Marshal(astObj.Export())
	if err != nil {
		return nil, fmt.Errorf("failed to serialize JS AST into JSON: %w", err)
	}

	var rawMessages []json.RawMessage
	if err := json.Unmarshal(rawJSON, &rawMessages); err != nil {
		return nil, fmt.Errorf("failed to parse scenarios array: %w", err)
	}

	// Deserialize the JSON into the Protobuf Scenario.
	var scenarios []*pb.Scenario
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true, // Ignore unknown fields for forward compatibility.
	}

	for _, rawMsg := range rawMessages {
		var scn pb.Scenario
		if err := unmarshaler.Unmarshal(rawMsg, &scn); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON into Protobuf Scenario: %w", err)
		}
		scenarios = append(scenarios, &scn)
	}

	return scenarios, nil
}
