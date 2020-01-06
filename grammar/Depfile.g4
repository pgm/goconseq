grammar Depfile;

all_declarations: ( declaration)*;

declaration: var_stmt | add_if_missing | rule_declaration;

/*
 # | rule # | include_stmt # | exec_profile # | remember_executed # | conditional # | eval_statement
 */

rule_declaration: 'rule' IDENTIFIER ':' input_bindings? output?;

input_bindings: 'inputs' ':' binding (',' binding)*;

binding: IDENTIFIER '=' json_obj;

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
