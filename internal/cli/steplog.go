package cli

import (
	"os"

	"dfl/internal/actions"
	"dfl/internal/runctx"
	"dfl/internal/setuplog"
)

func logStepStart(text string) {
	_ = setuplog.AppendStart(os.Getenv("DFL_LOG"), text)
}

func logStepEnd(status runctx.ResultStatus, message string) {
	_ = setuplog.AppendEnd(os.Getenv("DFL_LOG"), status, message)
}

func logStepResult(text string, status runctx.ResultStatus, message string, err error) {
	_ = setuplog.AppendResult(os.Getenv("DFL_LOG"), text, status, message, actions.OutputFromError(err))
}
