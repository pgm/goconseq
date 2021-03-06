An experimental rewrite of conseq in go.

[![Build Status](https://travis-ci.com/pgm/goconseq.svg?branch=master)](https://travis-ci.com/pgm/goconseq)

# Introduction

Conseq /kənˈsek/ is a "make"-like tool for assembling data transformation and processing workflows.

## Quickstart

To illustrate the simplest possible execution, we’ll describe creating running a single rule which writes "hello world" to a file.

1. create a file named "sample.conseq" containing

```
rule hello_world:
   outputs: {'type': 'output',
             'filename': filename('message.txt')}
   run "echo hello world > message.txt"
```

2. Run the script

```
$ conseq run sample.conseq
0 processes running (), 1 executions pending, 0 skipped

Summary of queue:
    state    transform      count  dirs
    -------  -----------  -------  ------
    pending  hello_world        1

Executing hello_world in state/r1 with inputs:
1 processes running (local-run:1), 0 executions pending, 0 skipped

Summary of queue:
    state      transform      count  dirs
    ---------  -----------  -------  --------
    local-run  hello_world        1  state/r1

Rule hello_world completed (state/r1). Results: {'outputs': [{'type': 'output', 'filename': {'$filename': 'message.txt'}}]}
Rule hello_world wrote the following files:
	state/r1/message.txt

0 processes running (), 0 executions pending, 0 skipped
1 jobs successfully executed

$
```

3. Now, we can list all artifacts and we’ll see our new artifact

```
$ conseq ls
For type=output:
  type    filename
  ------  -------------------------------------
  output  {'$filename': 'state/r1/message.txt'}
```

4. We can see the contents of the file by asking conseq to print the "filename" field the only artifact we have.

```
$ cat `conseq ls -f filename`
hello world
```

## Overview

While taking inspiration from tools such as [snakemake](https://snakemake.readthedocs.io/), [drake](https://github.com/Factual/drake) and [make](<https://en.wikipedia.org/wiki/Make_(software)>), conseq differs in that instead of rules depending on filename patterns, conseq rules consume and produce "artifacts".

These artifacts are essentially records with one or more named fields. Each field contain either a string for a value or a reference to a file. This richer data model allows users to describe the inputs of a rule with a relational query and simplifies the passing of multiple pieces of information to and from rules.

Conseq has three key concepts: "artifacts", "rules", and "applied rules" which are described in more detail below.

### Artifacts

Artifacts comprise of a set of key-value pairs. Values are either strings or file references.

Artifacts are generated as outputs from most rules, however, one can manually include artifacts in a conseq config file using an `artifact` statement. The artifact's fields are specified in syntax that is similar to python dictionaries.

_Example: artifact with two fields named "type" and "other":_

```
artifact {'type': 'sample', 'other': 'value'}
```

File references may refer to a local path, or a path to an object in google cloud storage (denoted with a "gs://" prefix such as `gsutil` uses). In either case, files are automatically transfered and localized to a path on the local filesystem before a rule is run. Thus, the `filename` annotation is needed so that conseq knows to treat this value as a file reference instead of a plain string.

_Example: An artifact which references a local file_

```
artifact {'type': 'bam', 'path': filename('file1.bam')}
```

_Example: An artifact which references a file in google cloud storage_

```
artifact {'type': 'bam', 'path': filename('gs://bucket/file2.bam')}
```

One might notice that all the examples above included a `type` field.

This is a convention that we commonly use, as we've found it's often easier to keep track of artifacts by adding a field named `type` and having all artifacts with the same value for `type` use the same field names. However, this is only a common convention, and conseq does not require this to be the case nor make any assumptions based on the value of `type`.

### Rules

A rule at minimium has a name and a query. In addition, rules typically will have one or more `run` statements describing scripts or commands which should be run when the rule executes. Whenever one or more new artifacts are found to satisfy the query, an a **applied rule** generated and the associated commands are executed.

_Example: a rule which executes `date` for every time an artifact with `type=sample` is found_

```
rule example_3:
  input: a={'type': 'sample'}
  run "date"
```

In addition to saying `run` which will simply execute the string via the bash shell, one can include scripts inline by using the syntax `run "...interpreter..." with "...script body..."`

_Example: a rule which runs a python script to print the time in seconds_

```
rule example_4:
  input: a={'type': 'sample'}
  run "python" with """
    import time
    print(time.time())
    """
```

In this example, the `"import time..."` block gets written to a temp file, and the command that actually actually runs is `python temp_file`. The use of "run ... with" is just convience syntax to make it possible to embed small scripts directly in the conseq config file and works well for a variety of scripting languages. (ie: `run "Rscript" with "..."`, `run "ruby" with "..."`, etc)

### Applied Rules

An applied rule is created when a query associated with a rule finds artifacts and binds them to the variables in the `input:` section of the rule. These variables are can be referenced within `run` statements.

_Example: a rule which runs a python script which prints the name from the input artifact_

```
artifact {'type': 'person', 'name': 'Joe'}
artifact {'type': 'person', 'name': 'Steve'}

rule hello_person:
  input: person={'type': 'person'}
  run "python" with """
    print("hello {{inputs.person.name}}")
    """
```

Now, in this example, we have a single rule, which will execute twice. It will execute once printing "hello Joe" and once printing "hello Steve".

There is a single rule, but since two artifacts are found, two **applied rules** are created. In each the `inputs.person` variable is bound to the artifact, and thus, we can get the name by referencing `inputs.person.name`.
