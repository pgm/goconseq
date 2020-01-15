package goconseq

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/pgm/goconseq/model"

	"github.com/stretchr/testify/assert"
)

type MockFiles struct {
}

func (m *MockFiles) EnsureLocallyAccessible(fileID int) (string, error) {
	return fmt.Sprintf("%d", fileID), nil
}

func (m *MockFiles) EnsureGloballyAccessible(fileID int) (string, error) {
	panic("unimp")
}

type CollectingListener struct {
	completed bool
	status    string
}

func (c *CollectingListener) Completed(state *model.CompletionState) {
	c.completed = true
}

func (c *CollectingListener) UpdateStatus(status string) {
	c.status = status
}

func TestLocalExec(t *testing.T) {
	mockFiles := &MockFiles{}

	jobDir, err := ioutil.TempDir("", "TestLocalExec")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(jobDir)

	breadcrumb := jobDir + "/breadcrumb"

	l := &LocalExec{
		files:  mockFiles,
		jobDir: jobDir}

	b := l.Builder(100)
	b.Prepare([]*model.RunWithStatement{&model.RunWithStatement{Executable: "date"},
		&model.RunWithStatement{Executable: "echo hello > " + breadcrumb}})

	// make sure the breadcrumb file does not exist
	_, err = os.Stat(breadcrumb)
	assert.True(t, os.IsNotExist(err))

	proc, err := b.Start(context.Background())
	assert.Nil(t, err)

	listener := &CollectingListener{}
	proc.Wait(listener)

	assert.True(t, listener.completed)

	// the breadcrumb file should exist now
	_, err = os.Stat(breadcrumb)
	assert.Nil(t, err)
}
