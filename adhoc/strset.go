package adhoc

type StrSet map[string]bool

func NewStrSet() StrSet {
	return make(StrSet)
}

func (ss StrSet) Add(s string) {
	ss[s] = true
}

func (ss StrSet) Len() int {
	return len(ss)
}

func (ss StrSet) In(s string) bool {
	return ss[s]
}
