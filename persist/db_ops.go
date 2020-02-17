package persist

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Envelope struct {
	Type string
	Body interface{}
}

type DeleteArtifactOp struct {
	ID int
}

func (op *DeleteArtifactOp) Update(db *DB) {
	delete(db.artifactHistoryByID, op.ID)
	delete(db.currentArtifacts, op.ID)
}

func (op *DeleteArtifactOp) GetType() string {
	return "DeleteArtifact"
}

type SetNextIDsOp struct {
	NextID            int
	NextAppliedRuleID int
}

func (op *SetNextIDsOp) Update(db *DB) {
	db.nextID = op.NextID
	db.nextAppliedRuleID = op.NextAppliedRuleID
}

func (op *SetNextIDsOp) GetType() string {
	return "SetNextIDs"
}

type SetFileOp struct {
	FileID     int
	LocalPath  string
	GlobalPath string
	SHA256     string
}

func (op *SetFileOp) Update(db *DB) {
	db.files[op.FileID] = &File{FileID: op.FileID,
		LocalPath:  op.LocalPath,
		GlobalPath: op.GlobalPath,
		SHA256:     op.SHA256}
}

func (op *SetFileOp) GetType() string {
	return "SetFile"
}

type SetArtifactOp struct {
	ID          int
	StringProps []*ArtifactStringProp
	FileProps   []*ArtifactFileProp
}

type ArtifactStringProp struct {
	Name  string
	Value string
}

type ArtifactFileProp struct {
	Name   string
	FileID int
}

func (op *SetArtifactOp) Update(db *DB) {
	props := NewArtifactProperties()

	for _, prop := range op.FileProps {
		props.Files[prop.Name] = prop.FileID
	}
	for _, prop := range op.StringProps {
		props.Strings[prop.Name] = prop.Value
	}

	artifact := Artifact{
		id:         op.ID,
		Properties: props}

	db.artifactHistoryByHash[artifact.Properties.Hash()] = &artifact
	db.artifactHistoryByID[artifact.id] = &artifact
}

func (op *SetArtifactOp) GetType() string {
	return "SetArtifact"
}

type SetAppliedRuleOp struct {
	ID          int
	Name        string
	Inputs      []*InputEntry
	Outputs     []int
	ResumeState string
	Hash        string
}

type InputEntry struct {
	Name      string
	Singleton bool
	Artifacts []int
}

func (op *SetAppliedRuleOp) Update(db *DB) {
	inputs := NewBindings()
	for _, input := range op.Inputs {
		artifacts := make([]*Artifact, len(input.Artifacts))
		for i, artifactID := range input.Artifacts {
			artifacts[i] = db.artifactHistoryByID[artifactID]
		}
		if input.Singleton {
			inputs.AddArtifact(input.Name, artifacts[0])
		} else {
			inputs.AddArtifacts(input.Name, artifacts)
		}
	}

	outputs := make([]*Artifact, len(op.Outputs))
	for i, artifactID := range op.Outputs {
		outputs[i] = db.artifactHistoryByID[artifactID]
	}

	appliedRule := AppliedRule{ID: op.ID,
		Name:        op.Name,
		Inputs:      inputs,
		Outputs:     outputs,
		ResumeState: op.ResumeState,
		Hash:        op.Hash}

	db.appliedRuleHistoryByID[appliedRule.ID] = &appliedRule
}

func (op *SetAppliedRuleOp) GetType() string {
	return "SetAppliedRule"
}

type DeleteAppliedRuleOp struct {
	ID int
}

func (op *DeleteAppliedRuleOp) Update(db *DB) {
	delete(db.currentAppliedRules, op.ID)
	delete(db.appliedRuleHistoryByID, op.ID)
}

func (op *DeleteAppliedRuleOp) GetType() string {
	return "DeleteAppliedRule"
}

func unmarshalAndCheck(msg json.RawMessage, op DBOp) (DBOp, error) {
	if err := json.Unmarshal(msg, op); err != nil {
		return nil, err
	}
	return op, nil
}

/////////////////////

func (w *OpLogWriter) WriteSetArtifact(artifact *Artifact) DBOp {
	stringProps := make([]*ArtifactStringProp, len(artifact.Properties.Strings))
	i := 0
	for k, s := range artifact.Properties.Strings {
		stringProps[i] = &ArtifactStringProp{Name: k, Value: s}
		i++
	}
	fileProps := make([]*ArtifactFileProp, len(artifact.Properties.Files))
	i = 0
	for k, s := range artifact.Properties.Files {
		fileProps[i] = &ArtifactFileProp{Name: k, FileID: s}
		i++
	}

	op := SetArtifactOp{
		ID:          artifact.id,
		StringProps: stringProps,
		FileProps:   fileProps}

	w.write(&op)

	return &op
}

func (w *OpLogWriter) WriteSetFile(file *File) DBOp {
	op := SetFileOp{
		FileID:     file.FileID,
		LocalPath:  file.LocalPath,
		GlobalPath: file.GlobalPath,
		SHA256:     file.SHA256}
	w.write(&op)

	return &op
}

func (w *OpLogWriter) WriteDeleteArtifact(artifactID int) DBOp {
	op := DeleteArtifactOp{ID: artifactID}
	w.write(&op)

	return &op
}

func (w *OpLogWriter) WriteSetAppliedRule(rule *AppliedRule) DBOp {
	inputs := make([]*InputEntry, len(rule.Inputs.ByName))
	i := 0
	for name, input := range rule.Inputs.ByName {
		srcArtifacts := input.GetArtifacts()
		artifacts := make([]int, len(srcArtifacts))
		_, singleton := input.(*SingleArtifact)
		for j, srcArtifact := range srcArtifacts {
			artifacts[j] = srcArtifact.id
		}
		inputs[i] = &InputEntry{
			Name:      name,
			Singleton: singleton,
			Artifacts: artifacts}
		i++
	}

	outputs := make([]int, len(rule.Outputs))
	for i, output := range rule.Outputs {
		outputs[i] = output.id
	}

	op := SetAppliedRuleOp{
		ID:          rule.ID,
		Name:        rule.Name,
		Inputs:      inputs,
		Outputs:     outputs,
		Hash:        rule.Hash,
		ResumeState: rule.ResumeState}

	w.write(&op)

	return &op
}

func (w *OpLogWriter) WriteSetNextIDs(nextID, nextAppRuleID int) DBOp {
	op := SetNextIDsOp{NextID: nextID, NextAppliedRuleID: nextAppRuleID}
	w.write(&op)
	return &op
}

func (w *OpLogWriter) WriteDeleteAppliedRule(id int) DBOp {
	op := DeleteAppliedRuleOp{ID: id}
	w.write(&op)
	return &op
}

func (w *OpLogWriter) Close() {
	err := w.file.Close()
	if err != nil {
		panic(err)
	}
}

func (w *OpLogWriter) Commit() {
	_, err := w.file.WriteString("commit\n")
	if err != nil {
		panic(err)
	}
}

func OpenLogReader(filename string) (*OpLogReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return &OpLogReader{reader: bufio.NewReader(file), file: file}, nil
}

func (l *OpLogReader) Close() error {
	return l.file.Close()
}

func (l *OpLogReader) ReadTransaction() ([]DBOp, error) {
	ops := make([]DBOp, 0, 10)
	for {
		// log.Printf("Attempting read %v", l.readCount)
		l.readCount += 1
		record, err := l.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}

		if string(record) == "commit\n" {
			break
		}

		op, err := unmarshalOp(record)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func unmarshalOp(input []byte) (DBOp, error) {
	var body json.RawMessage
	env := Envelope{
		Body: &body,
	}

	if err := json.Unmarshal(input, &env); err != nil {
		return nil, err
	}

	switch env.Type {
	case "DeleteAppliedRule":
		var op DeleteAppliedRuleOp
		return unmarshalAndCheck(body, &op)
	case "SetArtifact":
		var op SetArtifactOp
		return unmarshalAndCheck(body, &op)
	case "SetAppliedRule":
		var op SetAppliedRuleOp
		return unmarshalAndCheck(body, &op)
	case "DeleteArtifact":
		var op DeleteArtifactOp
		return unmarshalAndCheck(body, &op)
	case "SetFile":
		var op SetFileOp
		return unmarshalAndCheck(body, &op)
	case "SetNextIDs":
		var op SetNextIDsOp
		return unmarshalAndCheck(body, &op)
	default:
		return nil, fmt.Errorf("Unknown type: %s", env.Type)
	}

	panic("not reachable")
}

type OpLogWriter struct {
	file          *os.File
	disableWrites bool
}

func OpenLogWriter(filename string) (*OpLogWriter, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return &OpLogWriter{file: file}, nil
}

type OpLogReader struct {
	readCount int
	file      *os.File
	reader    *bufio.Reader
}

// all writes are not recoverable, so we panic instead of returning an error
func (w *OpLogWriter) write(x DBOp) {
	if w.disableWrites {
		panic("writes disabled")
	}
	env := Envelope{Type: x.GetType(), Body: x}
	buf, err := json.Marshal(&env)
	if err != nil {
		panic(err)
	}
	buf = append(buf, '\n')
	_, err = w.file.Write(buf)
	if err != nil {
		panic(err)
	}
}
