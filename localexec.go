package goconseq

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/pgm/goconseq/model"
	"github.com/pgm/goconseq/persist"
)

type Files interface {
	// given a fileId, return a path to that file which we can use locally
	EnsureLocallyAccessible(fileID int) (string, error)

	// given a fileId, return a globally accessible URL
	EnsureGloballyAccessible(fileID int) (string, error)
}

type LocalExec struct {
	files Files
}

type LocalProcess struct {
	process *os.Process
}

func (p *LocalProcess) Wait(listener model.Listener) {
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
	listener.Completed(&model.CompletionState{Success: true})
}

// Start a process.
func (e *LocalExec) Start(context context.Context, command []string, localizer model.Localizer) (model.Execution, error) {
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

func (e *LocalExec) Resume(resumeState string) (model.Execution, error) {
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

func (e *LocalExec) GetLocalizer() model.Localizer {
	return &LocalPathLocalizer{}
}

func (e *LocalExec) Localize(fileId int) (string, error) {
	path, err := e.files.EnsureLocallyAccessible(fileId)
	return path, err
}
