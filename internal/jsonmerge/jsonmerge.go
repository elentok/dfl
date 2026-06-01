// Package jsonmerge merges decoded JSON values with deep-merge object semantics
// and union (deduplicating) array semantics. It is pure: it operates on values
// produced by encoding/json and performs no I/O.
package jsonmerge

import (
	"bytes"
	"encoding/json"
	"maps"
	"reflect"
)

// Merge reduces the given decoded JSON values left to right. For each pair the
// rules are:
//
//   - both objects -> merge recursively
//   - both arrays  -> union: concatenate, deduplicate by deep equality, keep
//     first-seen order (applied at any depth)
//   - otherwise (scalar, array<->object, or any type mismatch) -> the later
//     value wins
//
// Inputs are expected to be the result of json.Unmarshal into an `any`
// (map[string]any, []any, or scalars). Merge does not mutate its inputs.
func Merge(values ...any) any {
	if len(values) == 0 {
		return nil
	}
	acc := values[0]
	for _, next := range values[1:] {
		acc = mergeTwo(acc, next)
	}
	return acc
}

func mergeTwo(a, b any) any {
	aObj, aOK := a.(map[string]any)
	bObj, bOK := b.(map[string]any)
	if aOK && bOK {
		return mergeObjects(aObj, bObj)
	}

	aArr, aOK := a.([]any)
	bArr, bOK := b.([]any)
	if aOK && bOK {
		return unionArrays(aArr, bArr)
	}

	// scalar, array<->object, or any type mismatch: later value wins.
	return b
}

func mergeObjects(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a)+len(b))
	maps.Copy(out, a)
	for k, v := range b {
		if existing, ok := out[k]; ok {
			out[k] = mergeTwo(existing, v)
		} else {
			out[k] = v
		}
	}
	return out
}

func unionArrays(a, b []any) []any {
	out := make([]any, 0, len(a)+len(b))
	for _, v := range a {
		if !containsDeepEqual(out, v) {
			out = append(out, v)
		}
	}
	for _, v := range b {
		if !containsDeepEqual(out, v) {
			out = append(out, v)
		}
	}
	return out
}

func containsDeepEqual(items []any, candidate any) bool {
	for _, item := range items {
		if reflect.DeepEqual(item, candidate) {
			return true
		}
	}
	return false
}

// Marshal renders a merged value canonically: object keys sorted, 2-space
// indent, and a trailing newline. The output is deterministic across runs.
func Marshal(value any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	// SetEscapeHTML(false) keeps characters like &, <, > intact, matching jq.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	// json.Encoder.Encode already appends a trailing newline. Object keys are
	// sorted by encoding/json when marshalling map[string]any.
	return buf.Bytes(), nil
}
