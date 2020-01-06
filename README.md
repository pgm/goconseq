random thoughts:
the use of "all" makes many things much more complicated. Is this really the best solution?
Instead of "all" bound vars, could we use "group by" metaphore? Generally like it, but it
does change the nature of how "all"s are evaluated. Each "all" is no longer independent.
Realistically, how often is that needed? I think the most common to use "all" as a way to
collect a single type of results, not multiple types.

Milestone 1: be able to run with no real persistance, only local executions, only foreach
Milestone 2: be able to run with real persistance, only local executions, only foreach
Milestone 3: be able to run with real persistance, run via delegate, only foreach

rule foo:
for_each: a, b where a.x = 'z' and a.y = 'z' and b.c = a.c
with_all: z where z.c = a.c
with_all: zz where zz.c = a.c
run "python ..." with "..."

interesting idea:
delgate which can read from GS
