grammar Depfile;

// tokens { IDENTIFIER, SHORT_STRING, LONG_STRING, NEWLINE, SPACES }

all_declarations: ( declaration)*;

declaration: var_stmt; //| add_if_missing;

/*
 # | rule # | include_stmt # | exec_profile # | remember_executed # | conditional # | eval_statement
 */
IDENTIFIER: [A-Za-z]+ [A-Za-z0-9_+-]*;

var_stmt: 'let' IDENTIFIER '=' quoted_string;

// different flavors of strings 

SHORT_STRING:
	'\'' (STRING_ESCAPE_SEQ | ~[\\\r\n\f'])* '\''
	| '"' ( STRING_ESCAPE_SEQ | ~[\\\r\n\f'])* '"';

LONG_STRING:
	'\'\'\'' LONG_STRING_ITEM*? '\'\'\''
	| '"""' LONG_STRING_ITEM*? '"""';

LONG_STRING_ITEM: LONG_STRING_CHAR | STRING_ESCAPE_SEQ;

LONG_STRING_CHAR: ~'\\';

STRING_ESCAPE_SEQ: '\\' . | '\\' NEWLINE;

quoted_string: LONG_STRING | SHORT_STRING;

COMMENT: '#' ~[\r\n\f]*;

SKIP_: ( COMMENT | SPACES) -> skip;

SPACES: [ \t]+;
NEWLINE: '\n';

add_if_missing: 'add-if-missing' json_obj;

json_obj:
	'{' json_name_value_pair (',' json_name_value_pair)* ','? '}';

json_name_value_pair: quoted_string ':' json_value;

json_value: quoted_string | json_obj | json_array;

json_array:
	'['; // json_array: '[' json_value ( ',' json_value)* ','? ']' | '[' ']';

