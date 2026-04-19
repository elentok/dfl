package cli

import (
	"os"

	"dfl/internal/runtime"
	"dfl/internal/runtimecmd"
	"dfl/internal/setuplog"
)

func logStepStart(text string) {
	_ = setuplog.AppendStart(os.Getenv("DFL_LOG"), text)
}

func logStepEnd(status runtime.ResultStatus, message string) {
	_ = setuplog.AppendEnd(os.Getenv("DFL_LOG"), status, message)
}

func logStepResult(text string, status runtime.ResultStatus, message string, err error) {
	_ = setuplog.AppendResult(os.Getenv("DFL_LOG"), text, status, message, runtimecmd.OutputFromError(err))
}
