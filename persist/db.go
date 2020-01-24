package persist

import (
	"fmt"
	"log"
	"os"
	"path"
)

// stored types: Artifacts, AppliedRules
// TODO: Make db threadsafe
const InitialStep = 0

type File struct {
	FileID     int
	LocalPath  string
	GlobalPath string
}

type DB struct {
	// todo: make DB methods threadsafe

	nextID            int
	nextAppliedRuleID int
	artifacts         map[int]*Artifact
	appliedRules      map[int]*AppliedRule
	files             map[int]*File
	stateDir          string
	writer            *OpLogWriter
}

type DBOp interface {
	Update(db *DB)

	GetType() string
}

func NewDB(stateDir string) *DB {
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		err = os.MkdirAll(stateDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	db := &DB{artifacts: make(map[int]*Artifact),
		appliedRules: make(map[int]*AppliedRule),
		stateDir:     stateDir}

	logPath := path.Join(stateDir, "db.journal")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		db.loadFromJournal(logPath)
	}

	writer, err := OpenLogWriter(logPath)
	if err != nil {
		panic(err)
	}
	db.writer = writer
	return db
}

func (db *DB) loadFromJournal(filename string) {
	reader, err := OpenLogReader(filename)
	if err != nil {
		panic(err)
	}
	for {
		ops, err := reader.ReadTransaction()
		if err != nil {
			log.Printf("Read log up to %v", err)
			break
		}
		for _, op := range ops {
			op.Update(db)
		}
	}
	reader.Close()
}

func (db *DB) Close() {
	db.writer.Close()
}

func (db *DB) GetNextApplicationID() int {
	ID := db.nextAppliedRuleID
	db.writer.WriteSetNextIDs(db.nextID, db.nextAppliedRuleID+1).Update(db)
	db.writer.Commit()
	return ID
}

func (db *DB) GetWorkDir(appliedRuleID int) string {
	return path.Join(db.stateDir,
		fmt.Sprintf("r%d", appliedRuleID))
}

// all _read_ operations do not return errors because they only use memory. All _write_ operations return an error
func (db *DB) FindAppliedRule(Name string, Inputs *Bindings) *AppliedRule {
	for _, appliedRule := range db.appliedRules {
		if appliedRule.IsEquivilent(Name, Inputs) {
			return appliedRule
		}
	}
	return nil
}

func (db *DB) GetAppliedRule(id int) *AppliedRule {
	return db.appliedRules[id]
}

func (db *DB) PersistAppliedRule(ID int, Name string, Inputs *Bindings, ResumeState string) (*AppliedRule, error) {
	appliedRule := &AppliedRule{id: ID, Name: Name, Inputs: Inputs, ResumeState: ResumeState}

	db.writer.WriteSetAppliedRule(appliedRule).Update(db)
	db.writer.Commit()

	return appliedRule, nil
}

func (db *DB) UpdateAppliedRuleComplete(ID int, Outputs []*Artifact) error {
	appliedRule := *db.appliedRules[ID]
	appliedRule.Outputs = Outputs
	appliedRule.ResumeState = ""

	db.writer.WriteSetAppliedRule(&appliedRule).Update(db)
	db.writer.Commit()

	return nil
}

func (db *DB) DeleteAppliedRule(ID int) error {
	db.writer.WriteDeleteAppliedRule(ID).Update(db)
	db.writer.Commit()

	return nil
}

// func (db *DB) DeleteArtifact(ID int) error {
// }

func (db *DB) PersistArtifact(ProducedBy int, Properties *ArtifactProperties) (*Artifact, error) {
	id := db.nextID
	artifact := &Artifact{id: id, ProducedBy: ProducedBy, Properties: Properties}

	db.writer.WriteSetNextIDs(db.nextID+1, db.nextAppliedRuleID).Update(db)
	db.writer.WriteSetArtifact(artifact).Update(db)
	db.writer.Commit()

	return artifact, nil
}

// func (db *DB) FindExactArtifact(Properties map[string]string) (*Artifact, error) {
// }

func (db *DB) FindArtifacts(Properties map[string]string) []*Artifact {
	results := make([]*Artifact, 0, 10)
	for _, artifact := range db.artifacts {
		if artifact.HasProperties(Properties) {
			results = append(results, artifact)
		}
	}
	return results
}

func (db *DB) AddFileGlobalPath(localPath string, globalPath string) *File {
	fileID := db.nextID
	file := &File{FileID: fileID, LocalPath: localPath, GlobalPath: globalPath}
	db.files[fileID] = file

	db.writer.WriteSetNextIDs(db.nextID+1, db.nextAppliedRuleID).Update(db)
	db.writer.WriteSetFile(file)
	db.writer.Commit()

	return file
}

func (db *DB) GetFile(fileID int) *File {
	return db.files[fileID]
}

func (db *DB) UpdateFile(fileID int, localPath string, globalPath string) *File {
	// never mutate, make a copy
	origFile := db.files[fileID]
	if localPath == "" {
		localPath = origFile.LocalPath
	}
	if globalPath == "" {
		globalPath = origFile.GlobalPath
	}
	file := &File{FileID: fileID, GlobalPath: globalPath, LocalPath: localPath}
	db.files[fileID] = file

	db.writer.WriteSetFile(file)
	db.writer.Commit()

	return file
}

// func (db *DB) FindAppliedRulesByName(name string) (*AppliedRule, error) {
// }

// func (db *DB) FindAppliedRulesByInput(id int) ([]*AppliedRule, error) {

// }

// func (db *DB) FindAppliedRulesByOutput(id int) ([]*AppliedRule, error) {

// }
