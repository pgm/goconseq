grammar Depfile;

all_declarations: ( declaration)* EOF;

declaration: var_stmt | add_if_missing | rule_declaration;

/*
 # | rule # | include_stmt # | exec_profile # | remember_executed # | conditional # | eval_statement
 */

rule_declaration:
	'rule' IDENTIFIER ':' input_bindings? output? run_statement*;

run_statement: 'run' quoted_string ('with' quoted_string)?;

input_bindings: 'inputs' ':' binding (',' binding)*;

filename_ref: 'filename' '(' quoted_string ')';

binding: IDENTIFIER '=' ALL? ( json_obj | filename_ref);

output: 'outputs' ':' json_obj (',' json_obj)*;

var_stmt: LET IDENTIFIER EQUALS quoted_string;

quoted_string: LONG_STRING | SHORT_STRING;

add_if_missing: 'add-if-missing' json_obj;

json_obj:
	'{' json_name_value_pair (',' json_name_value_pair)* ','? '}';

json_name_value_pair: quoted_string ':' json_value;

json_value: quoted_string; //| json_obj | json_array;

//json_array: '[' json_value ( ',' json_value)* ','? ']' | '[' ']';

////// lexer

LET: 'let';
ALL: 'all';

EQUALS: '=';

// different flavors of strings 
SHORT_STRING:
	'\'' (STRING_ESCAPE_SEQ | ~[\\\r\n\f'])* '\''
	| '"' ( STRING_ESCAPE_SEQ | ~[\\\r\n\f'])* '"';

LONG_STRING:
	'\'\'\'' LONG_STRING_ITEM*? '\'\'\''
	| '"""' LONG_STRING_ITEM*? '"""';

fragment LONG_STRING_ITEM: LONG_STRING_CHAR | STRING_ESCAPE_SEQ;

fragment LONG_STRING_CHAR: ~'\\';

fragment STRING_ESCAPE_SEQ: '\\' .;

COMMENT: '#' ~[\r\n\f]* -> channel(HIDDEN);

SPACES: [ \t\r\n]+ -> skip;

IDENTIFIER: [A-Za-z]+ [A-Za-z0-9_+-]*;
