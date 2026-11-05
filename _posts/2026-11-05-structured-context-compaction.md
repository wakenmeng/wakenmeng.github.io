---
title: "Can a Coding Agent Work in 16K of Context? Structured Compaction on SWE-bench Verified"
date: 2026-11-05
author: Waken
tags: [llm, agents, context, compaction, swe-bench, benchmark]
toc: true
---

Coding agents pile up history: every command, every file read, every test log. A recent paper, [You don't need long context](https://arxiv.org/abs/2611.05306), shows that frontier models lose almost nothing when they compact that history often and keep contexts short. I tried a different kind of compaction, a structured state the harness can check, and ran it on 100 SWE-bench Verified tasks under a 16K limit. It resolved **85%** of them for **$0.30 per task**, in line with the paper's results and cheaper than any setting in it.
<!-- more -->

## Background

The paper's method, end-of-context compaction (EoCC), is simple. When the context reaches a threshold, say 12K of a 16K limit, the model writes a text handoff and the handoff replaces the history. With GPT 5.6 Sol on all 500 Verified tasks, 16K/12K resolves 84.6% against 84.4% for the full, uncompacted history.

Each handoff is a full rewrite, though, and that leaves three gaps the paper itself points to:

- losses compound over repeated rewrites
- the model cannot tell what it forgot, so retrieval tools alone do not help
- nothing checks whether the handoff could actually replace the history

## Implementation

The idea is to treat compaction as maintaining a **Markov working state**: given the state and access to the repo, the next decision should not need the history.

- **Structured state, patched field by field.** Goal, facts, changes, checks, dead ends, plan and next action live in a JSON state. At each compaction the model submits a patch; the harness validates it and keeps every version. A rejected patch gets one retry, then falls back to a text handoff.
- **Evidence the harness can verify.** Facts and checks cite events in an append-only log kept outside the container. The harness hashes repo files after every action, so when a file changes it marks the facts that depend on it stale on its own. The model cannot revive a stale fact without a newer observation.
- **Pointers instead of copies.** Large outputs stay in the log, and the model reads bounded slices back with a `hist` tool when it needs them.
- **A hard budget.** 16K limit, compaction at 12K. Token estimates are anchored on the API's own counts, and tool outputs are trimmed when needed so the next compaction always fits.

It is built on mini-swe-agent 2.4.0 and scored with the official swebench harness on an x86 VM. The plan was pre-registered, the configuration was tuned on 20 separate tasks and frozen, and only then were the 100 formal tasks (a stratified sample of the 500) run once each.

## Results

| Setup | Context limit / threshold | Resolved | Cost per task | Tasks |
| --- | --- | --- | --- | --- |
| Paper: no compaction (full history) | none | 84.4% | $0.34 | 500 |
| Paper: EoCC | 48K / 36K | 88.8% | $0.37 | 500 |
| Paper: EoCC | 32K / 24K | 85.0% | $0.37 | 500 |
| Paper: EoCC | 24K / 18K | 85.0% | $0.38 | 500 |
| Paper: EoCC | 16K / 12K | 84.6% | $0.41 | 500 |
| **Structured compaction** | **16K, 24K for tasks that overflowed** | **85% (95% CI 77–91%)** | **$0.30** | 100 |
| Structured compaction | 16K only | 82% (95% CI 73–88%) | $0.27 | 100 |

Same model and list prices throughout; the paper's costs are recomputed from its token counts.

- **It holds accuracy at 16K.** 85%, the same band as the full history and the paper's 16K–32K settings.
- **It is the cheapest setup in the table**, because the state is patched instead of rewritten and contexts restart short.
- **It does not make the agent repeat itself.** Only 0.35% of actions after a compaction repeated earlier work.
- **16K alone is tight.** 8 tasks overflowed when a rejected patch had no room left for its retry. Rerun at 24K, all 8 completed and 6 resolved.

This is an informal, outside comparison: 100 tasks against the paper's 500, one run each, with the paper's own prompts and harness on its side. It shows parity, not an advantage. The whole formal run cost about $30 in API calls.

## Code and data

Everything is on GitHub: **[wakenmeng/context-compaction](https://github.com/wakenmeng/context-compaction)**. The repo has the agent and harness code, the pre-registered plan with every amendment, the per-task results, the full analysis, and the commands to reproduce it.
