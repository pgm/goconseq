package goconseq

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/pgm/goconseq/persist"
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

type LocalProcess struct {
	process *os.Process
}

func (p *LocalProcess) Wait(listener Listener) {
	listener.UpdateStatus("Executing")

	// attempt to wait directly, but this will fail if we're not the parent process
	p.process.Wait()

	// for {
	// 	err := p.process.Signal(syscall.Signal(0))
	// 	if err == nil {
	// 		// the process still exists and therefore is running
	// 	} else {
	// 		break
	// 	}

	// 	time.Sleep()
	// }

	log.Printf("todo: implement failure check")
	listener.Completed(&CompletionState{Success: true})
}

// Start a process.
func (e *LocalExec) Start(context context.Context, command []string, localizer Localizer) (Execution, error) {
	cmd := exec.CommandContext(context, command[0], command[1:]...)
	// cmd.Stdin = strings.NewReader("some input")
	// var out bytes.Buffer
	// cmd.Stdout = &out
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return &LocalProcess{
		process: cmd.Process}, err
}

func (e *LocalProcess) GetResumeState() string {
	return fmt.Sprintf("%d", e.process.Pid)
}

func (e *LocalExec) Resume(resumeState string) (Execution, error) {
	panic("unimp")
	// process, err := os.FindProcess()
	// if err != nil {
	// 	// according to docs, this should always succeed under unix
	// 	panic(err)
	// }
}

type LocalPathLocalizer struct {
	db *persist.DB
}

func (l *LocalPathLocalizer) Localize(fileID int) (string, error) {
	return l.db.GetFile(fileID).LocalPath, nil
}

func (e *LocalExec) GetLocalizer() Localizer {
	return &LocalPathLocalizer{}
}

func (e *LocalExec) Localize(fileId int) (string, error) {
	path, err := e.files.EnsureLocallyAccessible(fileId)
	return path, err
}
