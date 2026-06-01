package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"dfl/internal/jsonmerge"
	"dfl/internal/runctx"
)

func (o Runner) MergeJSON(ctx runctx.Context, componentRoot string, inputs []string, output string) (runctx.ResultStatus, string, error) {
	resolvedOutput, err := expandPath(output)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	values := make([]any, 0, len(inputs))
	for _, input := range inputs {
		resolvedInput, _, err := resolvePaths(componentRoot, input, output)
		if err != nil {
			return runctx.StatusFailed, "", err
		}
		data, err := os.ReadFile(resolvedInput)
		if err != nil {
			return runctx.StatusFailed, "", err
		}
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			return runctx.StatusFailed, "", fmt.Errorf("parsing %s: %w", resolvedInput, err)
		}
		values = append(values, value)
	}

	rendered, err := jsonmerge.Marshal(jsonmerge.Merge(values...))
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	current, err := os.ReadFile(resolvedOutput)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return runctx.StatusFailed, "", err
	}
	if bytes.Equal(current, rendered) {
		return runctx.StatusSkipped, "already up to date", nil
	}

	if ctx.DryRun {
		return runctx.StatusSuccess, fmt.Sprintf("would merge %d files into %s", len(inputs), resolvedOutput), nil
	}

	if err := os.MkdirAll(filepath.Dir(resolvedOutput), 0o755); err != nil {
		return runctx.StatusFailed, "", err
	}
	if err := os.WriteFile(resolvedOutput, rendered, 0o644); err != nil {
		return runctx.StatusFailed, "", err
	}

	return runctx.StatusSuccess, "done", nil
}
