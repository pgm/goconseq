package parser

import (
	"fmt"
	"log"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/pgm/goconseq/parser/antlrparser"
)

type Visitor struct {
	antlrparser.BaseDepfileVisitor
}

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

type Config struct {
	Vars map[string]string
}

type AddIfMissingStatement struct {
	Artifact map[string]string
}

func (s *AddIfMissingStatement) Eval(config *Config) error {
	panic("unimp")
}

type RuleStatement struct {
	Name    string
	Inputs  map[string]map[string]string
	Outputs []map[string]string
}

func (s *RuleStatement) Eval(config *Config) error {
	panic("unimp")
}

type LetStatement struct {
	Name  string
	Value string
}

func (s *LetStatement) Eval(config *Config) error {
	if existingValue, exists := config.Vars[s.Name]; exists {
		return fmt.Errorf("Cannot define %s as %s (already defined as %s)", s.Name, s.Value, existingValue)
	}
	config.Vars[s.Name] = s.Value
	return nil
}

type Statement interface {
	Eval(config *Config) error
}

func (s *Statements) Eval(config *Config) error {
	for _, stmt := range s.Statements {
		err := stmt.Eval(config)
		if err != nil {
			return err
		}
	}
	return nil
}

type Statements struct {
	Statements []Statement
}

func (s *Statements) Add(stmt Statement) {
	s.Statements = append(s.Statements, stmt)
}

func (v *Visitor) VisitQuoted_string(ctx *antlrparser.Quoted_stringContext) interface{} {
	log.Printf("VisitQuoted_string %v", ctx)
	for i, c := range ctx.GetChildren() {
		log.Printf("child %d: %v", i, c)
		log.Printf("payload: %v", c.GetPayload())
	}
	return "str"
}

func (v *Visitor) VisitVar_stmt(ctx *antlrparser.Var_stmtContext) interface{} {
	log.Printf("visiting %v", ctx)
	for i, c := range ctx.GetChildren() {
		log.Printf("child %d: %v", i, c)
		log.Printf("payload: %v", c.GetPayload())
	}
	return "ok"
}

func ParseString(s string) (*Statements, error) {
	is := antlr.NewInputStream(s)
	return parseCharStream(is)
}

// func parseCharStream(is antlr.CharStream) (*Statements, error) {
// 	lexer := antlrparser.NewDepfileLexer(is)
// 	// for i := 0; i < 5; i++ {
// 	// 	t := lexer.NextToken()
// 	// 	log.Printf("token %v, type=%d", t, t.GetTokenType())
// 	// }
// 	// panic("x")
// 	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
// 	p := antlrparser.NewDepfileParser(stream)
// 	tree := p.Var_stmt()
// 	// antlr.ParseTreeWalkerDefault.Walk(&Listener{}, )
// 	v := tree.Accept(&Visitor{})
// 	log.Printf("%v", v)
// 	return nil, v
// }

type CollectingErrorListener struct {
	*antlr.DefaultErrorListener
	errors *[]string
}

func (l *CollectingErrorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	*l.errors = append(*l.errors, msg)
}

func NewCollectingErrorListener(errors *[]string) *CollectingErrorListener {
	c := new(CollectingErrorListener)
	c.errors = errors
	return c
}

func parseCharStream(is antlr.CharStream) (*Statements, error) {
	errors := make([]string, 0)

	lexer := antlrparser.NewDepfileLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := antlrparser.NewDepfileParser(stream)

	// replace error handler with custom version which collects all errors
	p.RemoveErrorListeners()
	p.AddErrorListener(NewCollectingErrorListener(&errors))

	// perform parsing
	tree := p.All_declarations()

	// check to see if we got any errors in course of parsing
	if len(errors) > 0 {
		for _, errMsg := range errors {
			log.Printf(errMsg)
		}
		return nil, fmt.Errorf("%d errors", len(errors))
	}

	// if parsing was good, now try walk the CST to create statements
	statements := Statements{}
	l := Listener{Statements: &statements}
	antlr.ParseTreeWalkerDefault.Walk(&l, tree)

	return &statements, nil
}
