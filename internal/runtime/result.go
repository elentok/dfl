package runtime

type ResultStatus string

const (
	StatusSuccess ResultStatus = "success"
	StatusSkipped ResultStatus = "skipped"
	StatusFailed  ResultStatus = "failed"
)

type Result struct {
	Name     string
	Status   ResultStatus
	Message  string
	Children []Result
}
