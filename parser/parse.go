package parser

import (
	"./antlrparser"
	"github.com/antlr/antlr4/runtime/Go/antlr"
)

type Listener struct {
	antlrparser.BaseDepfileListener
}

func Parse() {
	is := antlr.NewInputStream("1 + 2 * 3")
	lexer := antlrparser.NewNewDepfileLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := antlrparser.NewCalcParser(stream)
	antlr.ParseTreeWalkerDefault.Walk(&Listener{}, p.Start())
}
