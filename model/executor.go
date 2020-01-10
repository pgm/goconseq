package model

import (
	"context"
	"os"
)

type NameValuePair struct {
	Name  string
	Value string
}

type CompletionState struct {
	Success bool

	// populated if !Success
	FailureMessage string
	FailureLogs    []*NameValuePair

	// non-nil only if process successfully started
	ProcessState *os.ProcessState
}

type Execution interface {
	GetResumeState() string
	// a blocking call which will wait until execution completes
	Wait(listener Listener)
}

type RunWith struct {
	Command string
	Body    string
}

type ExecutionBuilder interface {
	Localize(fileId int) (string, error)
	AddFile(body []byte) (string, error)

	Prepare(stmts []*RunWith) error

	Start(context context.Context) (exec Execution, err error)
}

type Executor interface {
	// Starts an execution
	Resume(resumeState string) (exec Execution, err error)
	Builder() ExecutionBuilder
}

// Listener is a set of callbacks that will be invoked over the lifespan of Start
type Listener interface {
	Completed(state *CompletionState)
	UpdateStatus(status string)
}
