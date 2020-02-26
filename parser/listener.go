package parser

import (
	"log"

	"github.com/pgm/goconseq/model"
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

func (l *Listener) AssertStackEmpty() {
	if len(l.Values) > 0 {
		panic("Internal parser error: stack not empty")
	}
}

func (l *Listener) PopQuery() *model.InputQuery {
	return l.Pop().(*model.InputQuery)
}

func (l *Listener) PopStrMap() map[string]string {
	return l.Pop().(map[string]string)
}

func (l *Listener) PopMap() map[string]interface{} {
	return l.Pop().(map[string]interface{})
}

func (l *Listener) PopArtifact() map[string]model.ArtifactValue {
	return l.Pop().(map[string]model.ArtifactValue)
}

func (l *Listener) PopArtifactValue() model.ArtifactValue {
	return l.Pop().(model.ArtifactValue)
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
	l.CurRule = &RuleStatement{Name: name, ExecutorName: model.DefaultExecutorName}
	l.Statements.Add(l.CurRule)
}

func (l *Listener) ExitRule_declaration(ctx *antlrparser.Rule_declarationContext) {
	l.CurRule = nil
}

func (l *Listener) ExitRun_statement(ctx *antlrparser.Run_statementContext) {
	executable := l.PopString()
	script := ""

	hasScript := ctx.Quoted_string(1) != nil
	if hasScript {
		script = l.PopString()
	}

	l.CurRule.RunStatements = append(l.CurRule.RunStatements, &model.RunWithStatement{Executable: executable, Script: script})
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

func (l *Listener) ExitArtifact_template_pair(ctx *antlrparser.Artifact_template_pairContext) {
	// pop and push the args to sanity check TOS
	value := l.PopString()
	name := l.PopString()

	l.Push(name)
	l.Push(value)
}

func (l *Listener) ExitArtifact_def_pair_value(ctx *antlrparser.Artifact_def_pair_valueContext) {
	if len(ctx.AllQuoted_string()) == 1 {
		// string on stack, push/pop it to check the type
		value := l.PopString()
		l.Push(model.ArtifactValue{Value: value})
	} else if len(ctx.AllQuoted_string()) == 2 {
		// this is probably "$filename" = filename,
		value := l.PopString()
		name := l.PopString()
		if name != "$filename" {
			log.Fatalf("expected $filename in Artifact_def_pair_value but got %s", name)
		}
		l.Push(model.ArtifactValue{Value: value, IsFilename: true})
	} else {
		panic("invalid Artifact_def_pair_value")
	}
}

func (l *Listener) ExitArtifact_template(ctx *antlrparser.Artifact_templateContext) {
	obj := make(map[string]string)
	i := 0
	for {
		pair := ctx.Artifact_template_pair(i)
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

func (l *Listener) ExitArtifact_def(ctx *antlrparser.Artifact_defContext) {
	obj := make(map[string]model.ArtifactValue)
	i := 0
	for {
		pair := ctx.Artifact_def_pair(i)
		if pair != nil {
			value := l.PopArtifactValue()
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

func mapFileRefArtifact(filename string) (map[string]string, map[string]model.ArtifactValue) {
	fileQuery := map[string]string{"name": filename, "type": model.FileRefType}
	fileArtifact := map[string]model.ArtifactValue{
		"name":     model.ArtifactValue{Value: filename},
		"filename": model.ArtifactValue{Value: filename, IsFilename: true},
		"type":     model.ArtifactValue{Value: model.FileRefType}}

	return fileQuery, fileArtifact
}

func (l *Listener) ExitBinding(ctx *antlrparser.BindingContext) {
	var value map[string]string

	isAll := ctx.ALL() != nil
	name := ctx.IDENTIFIER().GetText()
	if ctx.Artifact_template() != nil {
		value = l.PopStrMap()
	} else {
		// if not a json obj, then this is a filename ref
		if ctx.Filename_ref() == nil {
			panic("internal error")
		}
		filename := l.PopString()

		var fileArtifact map[string]model.ArtifactValue
		// query for finding file by filename
		value, fileArtifact = mapFileRefArtifact(filename)
		l.Statements.Add(&ArtifactStatement{fileArtifact})
	}

	l.Push(name)
	l.Push(&model.InputQuery{IsAll: isAll, Properties: value})
}

func (l *Listener) ExitInput_bindings(ctx *antlrparser.Input_bindingsContext) {
	bindings := make(map[string]*model.InputQuery)
	for _ = range ctx.AllBinding() {
		query := l.PopQuery()
		variable := l.PopString()
		bindings[variable] = query
	}
	//	l.Push(bindings)
	l.CurRule.Inputs = bindings
}

func (l *Listener) ExitOutput(ctx *antlrparser.OutputContext) {
	for _ = range ctx.AllArtifact_def() {
		outputArtifact := l.PopArtifact()
		output := RuleStatementOutput{}
		for k, v := range outputArtifact {
			output.Properties = append(output.Properties, UnresolvedOutputProperty{Name: k, IsFilename: v.IsFilename, Value: v.Value})
		}
		l.CurRule.Outputs = append(l.CurRule.Outputs, output)
	}
}

func (l *Listener) ExitFilename_ref(ctx *antlrparser.Filename_refContext) {
	// pop/push filename to make the parameter is there
	filename := l.PopString()
	l.Push(filename)
}

func (l *Listener) ExitAdd_if_missing(ctx *antlrparser.Add_if_missingContext) {
	artifact := l.PopArtifact()
	l.Statements.Add(&ArtifactStatement{artifact})
}
