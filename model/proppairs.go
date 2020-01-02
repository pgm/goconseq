package model

type PropPair struct {
	Name  string
	Value string
}

type PropPairs struct {
	//	pairs []*propPair
	pairSet map[PropPair]bool
}

func (pp *PropPairs) Add(pair PropPair) {
	if pp.pairSet == nil {
		pp.pairSet = make(map[PropPair]bool)
	}
	_, ok := pp.pairSet[pair]
	if !ok {
		pp.pairSet[pair] = true
	}
}

func (pp *PropPairs) Has(pair PropPair) bool {
	if pp.pairSet == nil {
		return false
	}
	return pp.pairSet[pair]
}

func (pp *PropPairs) Contains(other *PropPairs) bool {
	for pair, _ := range other.pairSet {
		if !pp.Has(pair) {
			return false
		}
	}
	return true
}
