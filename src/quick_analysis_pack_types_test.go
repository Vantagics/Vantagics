//go:build property_test

package main

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: datasource-pack-result-consistency, Property 9: PackStep 序列化往返一致性
// Validates: Requirements 8.4
//
// For any valid []PackStep list, marshaling to JSON and then unmarshaling back
// shall produce a list where each step has identical StepID, StepType, Code,
// Description, UserRequest, DependsOn, SourceTool, PairedSQLStepID, and
// EChartsConfigs values.

// genStepType generates a valid step type.
func genPackStepType() gopter.Gen {
	return gen.OneConstOf(stepTypeSQL, stepTypePython)
}

// genOptionalString generates a string that may be empty (simulating omitempty fields).
func genOptionalString() gopter.Gen {
	return gen.Frequency(
		map[int]gopter.Gen{
			3: gen.AlphaString(),
			1: gen.Const(""),
		},
	)
}

// genOptionalIntSlice generates an []int that may be nil or non-empty.
func genOptionalIntSlice() gopter.Gen {
	return gen.Frequency(
		map[int]gopter.Gen{
			1: gen.Const([]int(nil)),
			3: gen.SliceOfN(5, gen.IntRange(1, 100)).SuchThat(func(s []int) bool {
				return len(s) > 0
			}),
		},
	)
}

// genOptionalStringSlice generates a []string that may be nil or non-empty.
func genOptionalStringSlice() gopter.Gen {
	return gen.Frequency(
		map[int]gopter.Gen{
			1: gen.Const([]string(nil)),
			3: gen.SliceOfN(5, gen.AlphaString()).SuchThat(func(s []string) bool {
				return len(s) > 0
			}),
		},
	)
}

// genPackStep generates a random PackStep with all fields populated randomly.
// We split into two CombineGens calls to stay within gopter's limit.
func genPackStep() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(1, 1000),    // StepID
		genPackStepType(),        // StepType
		gen.AlphaString(),        // Code
		gen.AlphaString(),        // Description
		genOptionalString(),      // UserRequest
	).FlatMap(func(v interface{}) gopter.Gen {
		first := v.([]interface{})
		stepID := first[0].(int)
		stepType := first[1].(string)
		code := first[2].(string)
		description := first[3].(string)
		userRequest := first[4].(string)

		return gopter.CombineGens(
			genOptionalIntSlice(),    // DependsOn
			genOptionalString(),      // SourceTool
			gen.IntRange(0, 100),     // PairedSQLStepID
			genOptionalStringSlice(), // EChartsConfigs
		).Map(func(second []interface{}) PackStep {
			return PackStep{
				StepID:          stepID,
				StepType:        stepType,
				Code:            code,
				Description:     description,
				UserRequest:     userRequest,
				DependsOn:       second[0].([]int),
				SourceTool:      second[1].(string),
				PairedSQLStepID: second[2].(int),
				EChartsConfigs:  second[3].([]string),
			}
		})
	}, reflect.TypeOf(PackStep{}))
}

// genPackStepSlice generates a non-empty slice of random PackSteps.
func genPackStepSlice() gopter.Gen {
	return gen.SliceOfN(10, genPackStep()).SuchThat(func(steps []PackStep) bool {
		return len(steps) > 0
	})
}

// packStepsEqual compares two PackStep slices field by field, treating nil and
// empty slices as equivalent for omitempty JSON fields.
func packStepsEqual(a, b []PackStep) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].StepID != b[i].StepID {
			return false
		}
		if a[i].StepType != b[i].StepType {
			return false
		}
		if a[i].Code != b[i].Code {
			return false
		}
		if a[i].Description != b[i].Description {
			return false
		}
		if a[i].UserRequest != b[i].UserRequest {
			return false
		}
		if a[i].SourceTool != b[i].SourceTool {
			return false
		}
		if a[i].PairedSQLStepID != b[i].PairedSQLStepID {
			return false
		}
		// For omitempty slice fields, nil and empty slice are equivalent after JSON roundtrip
		if !intSlicesEquivalent(a[i].DependsOn, b[i].DependsOn) {
			return false
		}
		if !stringSlicesEquivalent(a[i].EChartsConfigs, b[i].EChartsConfigs) {
			return false
		}
	}
	return true
}

// intSlicesEquivalent treats nil and empty []int as equivalent.
func intSlicesEquivalent(a, b []int) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// stringSlicesEquivalent treats nil and empty []string as equivalent.
func stringSlicesEquivalent(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestProperty9_PackStepSerializationRoundtrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(time.Now().UnixNano())
	properties := gopter.NewProperties(parameters)

	properties.Property("PackStep list survives JSON marshal/unmarshal roundtrip", prop.ForAll(
		func(original []PackStep) bool {
			// Marshal to JSON
			data, err := json.Marshal(original)
			if err != nil {
				t.Logf("marshal error: %v", err)
				return false
			}

			// Unmarshal back
			var restored []PackStep
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Logf("unmarshal error: %v", err)
				return false
			}

			return packStepsEqual(original, restored)
		},
		genPackStepSlice(),
	))

	properties.TestingRun(t)
}
