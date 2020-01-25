package persist

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func verifyOp(t *testing.T, writeCallback func(*OpLogWriter), verifyCallback func([]DBOp)) {
	stateDir, err := ioutil.TempDir("", "TestDeleteArtifactOp")
	if err != nil {
		panic(err)
	}

	logPath := path.Join(stateDir, "log")
	w, err := OpenLogWriter(logPath)
	assert.Nil(t, err)
	writeCallback(w)
	w.Commit()
	w.Close()

	r, err := OpenLogReader(logPath)
	assert.Nil(t, err)
	ops, err := r.ReadTransaction()
	assert.Nil(t, err)
	verifyCallback(ops)
	r.Close()
}

func TestWriteSetFileOp(t *testing.T) {
	verifyOp(t, func(w *OpLogWriter) {
		w.WriteSetFile(&File{FileID: 12, LocalPath: "local", GlobalPath: "global"})
	}, func(ops []DBOp) {
		assert.Equal(t, 1, len(ops))
		op := ops[0].(*SetFileOp)
		assert.Equal(t, 12, op.FileID)
		assert.Equal(t, "global", op.GlobalPath)
		assert.Equal(t, "local", op.LocalPath)
	})
}

func TestWriteSetArtifact(t *testing.T) {
	props := NewArtifactProperties()
	props.Files["file"] = 200 // &File{FileID: 200}
	props.Strings["string"] = "value"
	artifact := &Artifact{
		id:         100,
		Properties: props}

	verifyOp(t, func(w *OpLogWriter) {
		w.WriteSetArtifact(artifact)
	}, func(ops []DBOp) {
		assert.Equal(t, 1, len(ops))
		op := ops[0].(*SetArtifactOp)
		assert.Equal(t, 100, op.ID)
		assert.Equal(t, 1, len(op.StringProps))
		assert.Equal(t, 1, len(op.FileProps))
	})
}

func makeArtifact(id int) *Artifact {
	return &Artifact{id: id}
}

func TestWriteSetAppliedRuleOp(t *testing.T) {
	props := NewArtifactProperties()
	props.Files["file"] = 200 // &File{FileID: 200}
	props.Strings["string"] = "value"
	// artifact := &Artifact{
	// 	id:         100,
	// 	Properties: props}

	bindings := NewBindings()
	bindings.AddArtifact("single", makeArtifact(41))
	bindings.AddArtifacts("multi", []*Artifact{makeArtifact(43), makeArtifact(44)})
	rule := &AppliedRule{
		ID:          30,
		Name:        "rule",
		Inputs:      bindings,
		Outputs:     []*Artifact{makeArtifact(40)},
		ResumeState: "resume"}

	verifyOp(t, func(w *OpLogWriter) {
		w.WriteSetAppliedRule(rule)
	}, func(ops []DBOp) {
		assert.Equal(t, 1, len(ops))
		op := ops[0].(*SetAppliedRuleOp)
		assert.Equal(t, 30, op.ID)
		assert.Equal(t, "rule", op.Name)
		assert.Equal(t, 1, len(op.Outputs))
		assert.Equal(t, 2, len(op.Inputs))
	})
}
