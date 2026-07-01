package actions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dfl/internal/runctx"
)

const (
	injectStartPrefix = "<!-- dfl:inject:start source="
	injectEndMarker   = "<!-- dfl:inject:end -->"
)

func (o Runner) Inject(ctx runctx.Context, componentRoot, source, target string, link bool) (runctx.ResultStatus, string, error) {
	resolvedSource, resolvedTarget, err := resolvePaths(componentRoot, source, target)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	sourceData, err := os.ReadFile(resolvedSource)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	targetData, err := os.ReadFile(resolvedTarget)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return runctx.StatusFailed, "", err
	}

	rendered, err := renderInjectedFile(string(targetData), string(sourceData), displayPath(resolvedSource), resolvedSource, link)
	if err != nil {
		return runctx.StatusFailed, "", err
	}

	if string(targetData) == rendered {
		return runctx.StatusSkipped, "already up to date", nil
	}

	if ctx.DryRun {
		if link {
			return runctx.StatusSuccess, fmt.Sprintf("would inject link %s into %s", resolvedSource, resolvedTarget), nil
		}
		return runctx.StatusSuccess, fmt.Sprintf("would inject %s into %s", resolvedSource, resolvedTarget), nil
	}

	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
		return runctx.StatusFailed, "", err
	}
	if err := os.WriteFile(resolvedTarget, []byte(rendered), targetMode(resolvedTarget, resolvedSource)); err != nil {
		return runctx.StatusFailed, "", err
	}

	return runctx.StatusSuccess, "done", nil
}

func renderInjectedFile(targetContent, sourceContent, sourcePath, resolvedSourcePath string, link bool) (string, error) {
	base, err := stripManagedInjection(targetContent)
	if err != nil {
		return "", err
	}

	base = strings.TrimRight(base, "\n")
	sourceContent = strings.TrimRight(sourceContent, "\n")
	payload := sourceContent
	if link {
		payload = "@" + displayPath(resolvedSourcePath)
	}

	var b strings.Builder
	if base != "" {
		b.WriteString(base)
		b.WriteString("\n\n")
	}
	b.WriteString(injectStartMarker(sourcePath))
	b.WriteString("\n")
	if payload != "" {
		b.WriteString(payload)
		b.WriteString("\n")
	}
	b.WriteString(injectEndMarker)
	b.WriteString("\n")
	return b.String(), nil
}

func stripManagedInjection(content string) (string, error) {
	start := strings.Index(content, injectStartPrefix)
	end := strings.Index(content, injectEndMarker)

	switch {
	case start == -1 && end == -1:
		return content, nil
	case start == -1 || end == -1:
		return "", errors.New("found incomplete injected block")
	case end < start:
		return "", errors.New("found invalid injected block ordering")
	}

	end += len(injectEndMarker)
	return strings.TrimRight(content[:start]+content[end:], "\n"), nil
}

func injectStartMarker(sourcePath string) string {
	return injectStartPrefix + sourcePath + " -->"
}
