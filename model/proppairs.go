package model

type StringProperty struct {
	Name  string
	Value string
}

type PropPairs struct {
	//	pairs []*propPair
	stringProps map[StringProperty]bool
	fileProps   map[string]int
}

func (pp *PropPairs) ensureInitialized() {
	if pp.stringProps == nil {
		pp.stringProps = make(map[StringProperty]bool)
		pp.fileProps = make(map[string]int)
	}
}

func (pp *PropPairs) AddFileProperty(name string, fileID int) {
	pp.ensureInitialized()
	pp.fileProps[name] = fileID
}

func (pp *PropPairs) AddStringProperty(name string, value string) {
	pp.ensureInitialized()
	pair := StringProperty{Name: name, Value: value}
	_, ok := pp.stringProps[pair]
	if !ok {
		pp.stringProps[pair] = true
	}
}

func (pp *PropPairs) GetFileProperties() []*FileProperty {
	pp.ensureInitialized()
	result := make([]*FileProperty, 0, len(pp.fileProps))
	for name, fileID := range pp.fileProps {
		result = append(result, &FileProperty{Name: name, FileID: fileID})
	}
	return result
}

func (pp *PropPairs) Has(name string, value string) bool {
	if pp.stringProps == nil {
		return false
	}
	return pp.stringProps[StringProperty{Name: name, Value: value}]
}

func (pp *PropPairs) Contains(other *PropPairs) bool {
	for pair, _ := range other.stringProps {
		if !pp.Has(pair.Name, pair.Value) {
			return false
		}
	}
	return true
}
