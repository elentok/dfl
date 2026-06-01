package jsonmerge

import (
	"encoding/json"
	"testing"
)

func decode(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("decode %q: %v", s, err)
	}
	return v
}

func mergeJSON(t *testing.T, inputs ...string) string {
	t.Helper()
	values := make([]any, len(inputs))
	for i, in := range inputs {
		values[i] = decode(t, in)
	}
	out, err := Marshal(Merge(values...))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(out)
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name   string
		inputs []string
		want   string // canonical (sorted keys, 2-space, trailing newline)
	}{
		{
			name:   "object deep merge",
			inputs: []string{`{"a":{"x":1}}`, `{"a":{"y":2},"b":3}`},
			want:   "{\n  \"a\": {\n    \"x\": 1,\n    \"y\": 2\n  },\n  \"b\": 3\n}\n",
		},
		{
			name:   "array union dedup",
			inputs: []string{`{"p":["a","b"]}`, `{"p":["b","c"]}`},
			want:   "{\n  \"p\": [\n    \"a\",\n    \"b\",\n    \"c\"\n  ]\n}\n",
		},
		{
			name:   "array union preserves first-seen order",
			inputs: []string{`["c","a"]`, `["a","b","c"]`},
			want:   "[\n  \"c\",\n  \"a\",\n  \"b\"\n]\n",
		},
		{
			name:   "nested arrays union at depth",
			inputs: []string{`{"o":{"p":[1,2]}}`, `{"o":{"p":[2,3]}}`},
			want:   "{\n  \"o\": {\n    \"p\": [\n      1,\n      2,\n      3\n    ]\n  }\n}\n",
		},
		{
			name:   "scalar last wins",
			inputs: []string{`{"k":1}`, `{"k":2}`},
			want:   "{\n  \"k\": 2\n}\n",
		},
		{
			name:   "array replaces object on type mismatch",
			inputs: []string{`{"k":{"x":1}}`, `{"k":[1,2]}`},
			want:   "{\n  \"k\": [\n    1,\n    2\n  ]\n}\n",
		},
		{
			name:   "object replaces array on type mismatch",
			inputs: []string{`{"k":[1,2]}`, `{"k":{"x":1}}`},
			want:   "{\n  \"k\": {\n    \"x\": 1\n  }\n}\n",
		},
		{
			name:   "three file reduce, last wins on scalar",
			inputs: []string{`{"k":1,"a":1}`, `{"k":2,"b":2}`, `{"k":3,"c":3}`},
			want:   "{\n  \"a\": 1,\n  \"b\": 2,\n  \"c\": 3,\n  \"k\": 3\n}\n",
		},
		{
			name:   "dedup object array elements",
			inputs: []string{`[{"id":1},{"id":2}]`, `[{"id":2},{"id":3}]`},
			want:   "[\n  {\n    \"id\": 1\n  },\n  {\n    \"id\": 2\n  },\n  {\n    \"id\": 3\n  }\n]\n",
		},
		{
			name:   "dedup number array elements",
			inputs: []string{`[1,2,3]`, `[2,3,4]`},
			want:   "[\n  1,\n  2,\n  3,\n  4\n]\n",
		},
		{
			name:   "keys sorted in output",
			inputs: []string{`{"z":1,"a":2,"m":3}`},
			want:   "{\n  \"a\": 2,\n  \"m\": 3,\n  \"z\": 1\n}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeJSON(t, tt.inputs...)
			if got != tt.want {
				t.Errorf("merge mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestMergeDoesNotMutateInputs(t *testing.T) {
	a := decode(t, `{"p":["a"]}`)
	b := decode(t, `{"p":["b"]}`)
	_ = Merge(a, b)

	first := a.(map[string]any)["p"].([]any)
	if len(first) != 1 {
		t.Fatalf("input a was mutated: %v", first)
	}
}

func TestMergeSingleValue(t *testing.T) {
	got := mergeJSON(t, `{"k":1}`)
	want := "{\n  \"k\": 1\n}\n"
	if got != want {
		t.Errorf("single value\n got: %q\nwant: %q", got, want)
	}
}
