# Ashveil Implementation Plans

Use [the project workflow](../AGENT_IMPLEMENTATION_WORKFLOW.md). No plugin is
required. Keep active plans here and design records in `../designs/`.

A plan names the files and behavior each task changes, its tests (including
real integration entry points), and its completion criteria. Player-facing
phases include help, keyword indexing, tutorial pointers, and rendering tests.

Work in the isolated feature workspace required by `AGENTS.md`. Run focused
checks per task, then independent phase review and final verification. Do not
repeat a full baseline test run for a new branch. Record the final review and
verification in `docs/PROJECT_STATUS.md` before integrating the phase.

Completed checklists can be removed once their useful decisions are captured
in the retained design, package guidance, and status. They remain in git
history. The retained 32b plan has an outstanding review note in project status;
its presence does not mean its implementation should be repeated.
