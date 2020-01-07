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

type Localizer interface {
	Localize(fileId int) (string, error)
}

type Executor interface {
	// Starts an execution
	Start(context context.Context, command []string, localizer Localizer) (exec Execution, err error)
	Resume(resumeState string) (exec Execution, err error)
	GetLocalizer() Localizer
}

// Listener is a set of callbacks that will be invoked over the lifespan of Start
type Listener interface {
	Completed(state *CompletionState)
	UpdateStatus(status string)
}
