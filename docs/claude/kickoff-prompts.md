# Kickoff prompts

Reusable prompts for each phase. Saved here as part of the process documentation.

## Shared note (append to every prompt below)
> Also: keep `docs/ai-usage-log.md` up to date. If this session produces a prompt or technique that shows novel, unique, or advanced use of AI tools (clever constraints, self-verification, role separation, effective use of plan mode/subagents/context management, or a failure I corrected), flag it as a "Log candidate" with a one-line reason and, once I confirm, append an entry using the template in that file. Be selective; quality over quantity.

## 1. Designer (run in claude.ai chat, right after the design session)
> Here is the take-home prompt and my notes from the design session with their engineer.
> [paste prompt] [paste notes]
> Produce, in this order: (1) `SPEC.md` with agreed deliverables, numbered behavior rules, assumptions, and open questions; (2) `DESIGN.md`, at most 2 pages: stack, modules, data model, key interfaces, testing strategy; (3) `PLAN.md`: vertical slices ordered so the thinnest end-to-end version ships first, each with acceptance criteria and tests. Use the templates provided. Be ruthless about scope: we have about 2 hours of build time. List anything you cut under "Future work". Ask me up to 5 clarifying questions first if anything is ambiguous. Also follow the shared AI-usage-log note above.

## 2. Implementor (fresh Claude Code session per slice or two)
> Read CLAUDE.md, SPEC.md, DESIGN.md, PLAN.md, and DECISIONS.md. Implement the next unchecked slice in PLAN.md using tests first. Follow the workflow in CLAUDE.md. Before coding, summarize your approach in 2-3 lines. When done, run all tests, commit, tick off the slice, and log any deviations in DECISIONS.md. Also follow the shared AI-usage-log note above.

## 3. Reviewer (fresh session, around 2:30 and again near 3:30)
> You are reviewing, not building. Read SPEC.md, DESIGN.md, PLAN.md, then the code. Report: (1) SPEC requirements not met or only partially met; (2) bugs and unhandled edge cases; (3) missing or weak tests; (4) dead code or unnecessary abstractions; (5) README accuracy. Don't change code; produce a prioritized list with file references. Then do a fresh-clone check: clone into a temp dir and follow only the README to install, run, and test. Also follow the shared AI-usage-log note above.

## 4. Walkthrough prep (final 30 minutes)
> Based on the repo, SPEC.md, and DECISIONS.md, draft: a 5-minute demo script, the 5 most important product decisions with tradeoffs, the key abstractions and why, what I'd do with another day, and likely questions an interviewer would ask with suggested answers. Also review `docs/ai-usage-log.md`: rank the entries, fill gaps, and draft a 60-second talking point on how I used AI as a core part of my process.
