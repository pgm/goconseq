package parser

import (
	"fmt"
	"log"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/pgm/goconseq/parser/antlrparser"
)

// type Visitor struct {
// 	antlrparser.BaseDepfileVisitor
// }

type Config struct {
	Vars map[string]string
}

// func (v *Visitor) VisitQuoted_string(ctx *antlrparser.Quoted_stringContext) interface{} {
// 	log.Printf("VisitQuoted_string %v", ctx)
// 	for i, c := range ctx.GetChildren() {
// 		log.Printf("child %d: %v", i, c)
// 		log.Printf("payload: %v", c.GetPayload())
// 	}
// 	return "str"
// }

// func (v *Visitor) VisitVar_stmt(ctx *antlrparser.Var_stmtContext) interface{} {
// 	log.Printf("visiting %v", ctx)
// 	for i, c := range ctx.GetChildren() {
// 		log.Printf("child %d: %v", i, c)
// 		log.Printf("payload: %v", c.GetPayload())
// 	}
// 	return "ok"
// }

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
