package goconseq

import (
	"context"
	"os"
	"os/exec"
)

type NameValuePair struct {
	Name  string
	Value string
}

type Files interface {
	// given a fileId, return a path to that file which we can use locally
	EnsureLocallyAccessible(fileID int) (string, error)

	// given a fileId, return a globally accessible URL
	EnsureGloballyAccessible(fileID int) (string, error)
}

type LocalExec struct {
	files Files
}

type CompletionState struct {
	Success bool

	// populated if !Success
	FailureMessage string
	FailureLogs    []*NameValuePair

	// non-nil only if process successfully started
	ProcessState *os.ProcessState
}

// Listener is a set of callbacks that will be invoked over the lifespan of Start
type Listener interface {
	Started(resumeState string)
	Completed(state *CompletionState)
	UpdateStatus(status string)
}

// Start a process. Blocks until execution completes. Intended to be invoked in own goroutine
func (e *LocalExec) Start(context context.Context, command []string, listener Listener) {
	cmd := exec.CommandContext(context, command[0], command[1:]...)
	// cmd.Stdin = strings.NewReader("some input")
	// var out bytes.Buffer
	// cmd.Stdout = &out
	listener.UpdateStatus("Executing")
	err := cmd.Start()
	if err != nil {
		listener.Completed(&CompletionState{Success: false, FailureMessage: err.Error()})
	} else {
		err = cmd.Wait()
		listener.Completed(&CompletionState{Success: true, ProcessState: cmd.ProcessState})
	}
}

func (e *LocalExec) Localize(fileId int) (string, error) {
	path, err := e.files.EnsureLocallyAccessible(fileId)
	return path, err
}

func (e *LocalExec) Resume(context context.Context, resumeState string, listener Listener) {
	panic("unimp")
}
