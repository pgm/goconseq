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

binding: IDENTIFIER '=' ALL? ( artifact_template | filename_ref);

output: 'outputs' ':' artifact_def (',' artifact_def)*;

var_stmt: LET IDENTIFIER EQUALS quoted_string;

quoted_string: LONG_STRING | SHORT_STRING;

add_if_missing: 'add-if-missing' artifact_def;

artifact_template:
	'{' artifact_template_pair (',' artifact_template_pair)* ','? '}';

artifact_template_pair: quoted_string ':' quoted_string;

result_outputs: '[' artifact_def (',' artifact_def)* ','? ']';

artifact_def:
	'{' artifact_def_pair (',' artifact_def_pair)* ','? '}';

artifact_def_pair: quoted_string ':' artifact_def_pair_value;

artifact_def_pair_value:
	quoted_string
	| '{' quoted_string ':' quoted_string '}';

json_value: quoted_string;

////// lexer

LET: 'let';
ALL: 'all';
EQUALS: '=';

// different flavors of strings 
SHORT_STRING:
	'\'' (STRING_ESCAPE_SEQ | ~[\\\r\n\f'])* '\''
	| '"' ( STRING_ESCAPE_SEQ | ~[\\\r\n\f"])* '"';

LONG_STRING:
	'\'\'\'' LONG_STRING_ITEM* '\'\'\''
	| '"""' LONG_STRING_ITEM* '"""';

fragment LONG_STRING_ITEM: LONG_STRING_CHAR | STRING_ESCAPE_SEQ;

fragment LONG_STRING_CHAR: ~'\\';

fragment STRING_ESCAPE_SEQ: '\\' .;

COMMENT: '#' ~[\r\n\f]* -> channel(HIDDEN);

SPACES: [ \t\r\n]+ -> skip;

IDENTIFIER: [A-Za-z]+ [A-Za-z0-9_+-]*;
