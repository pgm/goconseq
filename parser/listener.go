package parser

import (
	"github.com/pgm/goconseq/parser/antlrparser"
)

type Listener struct {
	antlrparser.BaseDepfileListener
	Statements *Statements
	Values     []interface{}
	CurRule    *RuleStatement
}

func (l *Listener) Pop() interface{} {
	last := len(l.Values) - 1
	v := l.Values[last]
	l.Values = l.Values[:last]
	return v
}

func (l *Listener) PopString() string {
	return l.Pop().(string)
}

func (l *Listener) PopStrMap() map[string]string {
	return l.Pop().(map[string]string)
}

func (l *Listener) Push(value interface{}) {
	l.Values = append(l.Values, value)
}

// func (s *Listener) VisitErrorNode(node antlr.ErrorNode) {
// 	log.Printf("error %v", node)
// 	pp := node.GetPayload()
// 	log.Printf("error %v", pp)
// }
func parseTripleQuotedString(s string) string {
	// todo handle escaping
	return s[3 : len(s)-3]
}

func parseQuotedString(s string) string {
	// todo handle escaping
	return s[1 : len(s)-1]
}

func (l *Listener) EnterRule_declaration(ctx *antlrparser.Rule_declarationContext) {
	name := ctx.IDENTIFIER().GetText()
	l.CurRule = &RuleStatement{Name: name}
	l.Statements.Add(l.CurRule)
}

func (l *Listener) ExitRule_declaration(ctx *antlrparser.Rule_declarationContext) {
	l.CurRule = nil
}

func (l *Listener) ExitQuoted_string(ctx *antlrparser.Quoted_stringContext) {
	t := ctx.LONG_STRING()
	if t != nil {
		l.Push(parseTripleQuotedString(t.GetText()))
	} else {
		t = ctx.SHORT_STRING()
		l.Push(parseQuotedString(t.GetText()))
	}
}

func (l *Listener) ExitJson_name_value_pair(ctx *antlrparser.Json_name_value_pairContext) {
	// pop and push the args to sanity check TOS
	value := l.PopString()
	name := l.PopString()

	l.Push(name)
	l.Push(value)
}

func (l *Listener) ExitJson_obj(ctx *antlrparser.Json_objContext) {
	obj := make(map[string]string)
	i := 0
	for {
		pair := ctx.Json_name_value_pair(i)
		if pair != nil {
			value := l.PopString()
			name := l.PopString()
			obj[name] = value
		} else {
			break
		}
		i++
	}
	l.Push(obj)
}

func (l *Listener) ExitVar_stmt(ctx *antlrparser.Var_stmtContext) {
	name := ctx.IDENTIFIER().GetText()
	//name := ctx.GetChild(1).GetPayload().(antlr.Token).GetText()
	//	value := ctx.GetChild(3).GetPayload().(antlr.ParseTree).GetText()
	// value := ctx.Quoted_string().GetText()
	value := l.PopString()
	// for i, c := range ctx.GetChildren() {
	// 	log.Printf("child %d: %v", i, c)
	// 	p := c.GetPayload()
	// 	tn := p.(antlr.Token)
	// 	txt := tn.GetText()
	// 	log.Printf("text: %s", txt)
	// }
	//	pp := ctx.GetChild(1).GetPayload()
	l.Statements.Add(&LetStatement{Name: name, Value: value})
}

func (l *Listener) ExitBinding(ctx *antlrparser.BindingContext) {
	name := ctx.IDENTIFIER().GetText()
	value := l.PopStrMap()

	l.Push(name)
	l.Push(value)
}

func (l *Listener) ExitInput_bindings(ctx *antlrparser.Input_bindingsContext) {
	bindings := make(map[string]map[string]string)
	for _ = range ctx.AllBinding() {
		query := l.PopStrMap()
		variable := l.PopString()
		bindings[variable] = query
	}
	l.Push(bindings)
}

func (l *Listener) ExitAdd_if_missing(ctx *antlrparser.Add_if_missingContext) {
	artifact := l.PopStrMap()
	l.Statements.Add(&AddIfMissingStatement{Artifact: artifact})
}

func (l *Listener) ExitOutput(ctx *antlrparser.OutputContext) {
	for _ = range ctx.AllJson_obj() {
		output := l.PopStrMap()
		l.CurRule.Outputs = append(l.CurRule.Outputs, output)
	}
}
