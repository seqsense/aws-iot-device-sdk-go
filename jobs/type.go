package jobs

type JobExecutionState string

const (
	Queued     JobExecutionState = "QUEUED"
	InProgress JobExecutionState = "IN_PROGRESS"
	Failed     JobExecutionState = "FAILED"
	Succeeded  JobExecutionState = "SUCCEEDED"
	Canceled   JobExecutionState = "CANCELED"
	TimedOut   JobExecutionState = "TIMED_OUT"
	Rejected   JobExecutionState = "REJECTED"
	Removed    JobExecutionState = "REMOVED"
)

type JobExecutionSummary struct {
	JobID           string `json:"jobId"`
	QueuedAt        int64  `json:"queuedAt"`
	StartedAt       int64  `json:"startedAt"`
	LastUpdatedAt   int64  `json:"lastUpdatedAt"`
	VersionNumber   int    `json:"versionNumber"`
	ExecutionNumber int    `json:"executionNumber"`
}

type JobExecution struct {
	JobID           string            `json:"jobId"`
	ThingName       string            `json:"thingName"`
	JobDocument     interface{}       `json:"jobDocument"`
	Status          JobExecutionState `json:"status"`
	StatusDetails   map[string]string `json:"statusDetails"`
	QueuedAt        int64             `json:"queuedAt"`
	StartedAt       int64             `json:"startedAt"`
	LastUpdatedAt   int64             `json:"lastUpdatedAt"`
	VersionNumber   int               `json:"versionNumber"`
	ExecutionNumber int               `json:"executionNumber"`
}

type ErrorResponse struct {
	Code           string            `json:"code"`
	Message        string            `json:"message"`
	ClientToken    string            `json:"clientToken"`
	Timestamp      int64             `json:"timestamp"`
	ExecutionState JobExecutionState `json:"executionState"`
}

func (e *ErrorResponse) Error() string {
	return e.Code + ": " + e.Message
}

type jobExecutionsChangedMessage struct {
	Jobs      map[JobExecutionState][]JobExecutionSummary `json:"jobs"`
	Timestamp int64                                       `json:"timestamp"`
}

type simpleRequest struct {
	ClientToken string `json:"clientToken"`
}

type getPendingJobExecutionsResponse struct {
	InProgressJobs []JobExecutionSummary `json:"inProgressJobs"`
	QueuedJobs     []JobExecutionSummary `json:"queuedJobs"`
	Timestamp      int64                 `json:"timestamp"`
	ClientToken    string                `json:"clientToken"`
}

type describeJobExecutionRequest struct {
	ExecutionNumber    int    `json:"executionNumber,omitempty"`
	IncludeJobDocument bool   `json:"includeJobDocument"`
	ClientToken        string `json:"clientToken"`
}

type describeJobExecutionResponse struct {
	Execution   JobExecution `json:"execution"`
	Timestamp   int64        `json:"timestamp"`
	ClientToken string       `json:"clientToken"`
}
