package executor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pgm/goconseq/model"
)

type Files interface {
	// given a fileId, return a path to that file which we can use locally
	EnsureLocallyAccessible(fileID int) (string, error)

	// given a fileId, return a globally accessible URL
	EnsureGloballyAccessible(fileID int) (string, error)
}

type LocalExec struct {
	Files  Files
	JobDir string
}

type LocalExecBuilder struct {
	workDir string

	command []string

	files     Files
	fileCount int
}

type LocalChildProcess struct {
	workDir string
	process *os.Process
}

func (p *LocalChildProcess) Wait(listener model.Listener) {
	listener.UpdateStatus("Executing")

	// attempt to wait directly, but this will fail if we're not the parent process
	state, err := p.process.Wait()

	if err != nil {
		panic(err)
	}

	rusage := state.SysUsage().(*syscall.Rusage)
	maxRSSInMB := float64(rusage.Maxrss) / (1024 * 1024) // convert to MB
	utime := float64(rusage.Utime.Sec) + (float64(rusage.Utime.Usec) / 1000000)
	stime := float64(rusage.Stime.Sec) + (float64(rusage.Stime.Usec) / 1000000)
	log.Printf("%s: PID %d terminated with exit code %d, max RSS: %.1f (MB), utime: %.1f (sec), stime: %.1f (sec)", p.workDir, p.process.Pid, state.ExitCode(), maxRSSInMB, utime, stime)

	if state.Success() {
		listener.Completed(&model.CompletionState{Success: true})
	} else {
		logs := []*model.NameValuePair{&model.NameValuePair{Name: "stdout", Value: p.workDir + "/stdout.txt"},
			&model.NameValuePair{Name: "stderr", Value: p.workDir + "/stderr.txt"}}
		listener.Completed(&model.CompletionState{Success: false,
			FailureMessage: fmt.Sprintf("Exit code was non-zero: %d", state.ExitCode()),
			FailureLogs:    logs})
	}
}

type LocalOtherProcess struct {
	process *os.Process
}

func (p *LocalOtherProcess) Wait(listener model.Listener) {
	sleepDuration := 10 * time.Millisecond
	MaxSleepDuration := 5 * time.Second

	for {
		err := p.process.Signal(syscall.Signal(0))
		if err == nil {
			// the process still exists and therefore is running
		} else {
			break
		}

		time.Sleep(sleepDuration)

		// exponentionally sleep for 1/3 longer, with an upper bound
		sleepDuration = sleepDuration * 4 / 3
		if sleepDuration > MaxSleepDuration {
			sleepDuration = MaxSleepDuration
		}
	}

	log.Printf("todo: implement failure check")
	listener.Completed(&model.CompletionState{Success: true})
}

func (e *LocalExecBuilder) Prepare(runStatements []*model.RunWithStatement) error {
	var sb strings.Builder

	sb.WriteString("set -ex\n")
	sb.WriteString("EXIT_STATUS=0\n")
	sb.WriteString("rm -f result.json\n")

	for _, runStatement := range runStatements {
		if strings.HasPrefix(runStatement.Executable, "cat > ") && runStatement.Script == "" {
			log.Printf("Warning: TODO: remove hack for cat")
			sb.WriteString(runStatement.Executable)
		} else {
			sb.WriteString(`if [ $EXIT_STATUS == 0 ]; then
			# Propagate kill if shell receives SIGTERM or SIGINT
			trap 'kill -TERM $PID' TERM INT
		  `)
			if runStatement.Script != "" {
				localName, err := e.AddFile([]byte(runStatement.Script))
				if err != nil {
					return err
				}
				sb.WriteString("  " + runStatement.Executable + " " + localName + " &\n")
			} else {
				sb.WriteString("  " + runStatement.Executable + " &\n")
			}
			sb.WriteString(`  PID=$!
			wait $PID
			trap - TERM INT
			wait $PID
			EXIT_STATUS=$?
		  fi
		  `)
		}
	}

	sb.WriteString("exit $EXIT_STATUS\n")
	scriptName, err := e.AddFile([]byte(sb.String()))
	if err != nil {
		return err
	}

	e.command = []string{"bash", scriptName}

	return nil
}

// Start a process.
func (e *LocalExecBuilder) Start(context context.Context) (model.Execution, error) {
	cmd := exec.CommandContext(context, e.command[0], e.command[1:]...)
	cmd.Dir = e.workDir

	stdoutFile, err := os.Create(e.workDir + "/stdout.txt")
	if err != nil {
		panic(err)
	}
	defer stdoutFile.Close()
	cmd.Stdout = stdoutFile

	stderrFile, err := os.Create(e.workDir + "/stderr.txt")
	if err != nil {
		panic(err)
	}
	defer stderrFile.Close()
	cmd.Stderr = stderrFile

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return &LocalChildProcess{
		workDir: e.workDir,
		process: cmd.Process}, nil
}

func (e *LocalChildProcess) GetResumeState() string {
	return fmt.Sprintf("%d", e.process.Pid)
}

func (e *LocalExec) Resume(resumeState string) (model.Execution, error) {
	pid, err := strconv.Atoi(resumeState)
	if err != nil {
		panic(err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		// according to docs, this should always succeed under unix
		panic(err)
	}

	return &LocalOtherProcess{process: process}, nil
}

func (e *LocalOtherProcess) GetResumeState() string {
	return fmt.Sprintf("%d", e.process.Pid)
}

func (l *LocalExecBuilder) Localize(fileID int) (string, error) {
	return l.files.EnsureLocallyAccessible(fileID)
}

func (e *LocalExec) Builder(jobIndex int) model.ExecutionBuilder {
	workDir := e.JobDir + "/r" + strconv.Itoa(jobIndex)
	os.MkdirAll(workDir, os.ModePerm)
	return &LocalExecBuilder{
		workDir: workDir,
		files:   e.Files}
}

// // func (e *LocalExec) Localize(fileId int) (string, error) {
// // 	path, err := e.files.EnsureLocallyAccessible(fileId)
// // 	return path, err
// }

func (e *LocalExecBuilder) AddFile(body []byte) (string, error) {
	//	conseqFiles := e.workDir+"/conseqfiles"
	_ = os.Mkdir(e.workDir+"/conseqfiles", os.ModePerm)

	e.fileCount++
	filename := fmt.Sprintf("conseqfiles/file%d", e.fileCount)

	file, err := os.Create(e.workDir + "/" + filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.Write(body)

	return filename, err
}
