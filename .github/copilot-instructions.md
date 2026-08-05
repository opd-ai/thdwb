# Go Coding Backlog Rules (Nemotron/Qwen Optimized)

## 🚨 Rate Limit Guardrails (Anti-429)
- NEVER dump unmodified files or massive directory readouts into chat. Keep tool outputs highly scoped.
- If a 429 API rate limit error occurs, pause execution for 5 seconds and retry.

## 📋 Task Priority & Loop Rules
- Process backlog sequentially from the first unchecked `- [ ]` item.
- Strict File Hierarchy: `AUDIT.md` ➔ `PLAN.md` ➔ `ROADMAP.md`. Do not skip ahead.
- Mark completed tasks immediately with `- [x]`.

## 🛠️ Implementation & Quality Standards
- **Function Metrics**: Maximum 30 lines per function. Maximum 10 cyclomatic complexity.
- **Documentation**: Minimum 70% doc coverage. Match local error wrapping and API signatures.
- **Tests**: Run `go test -race ./...` and `go vet ./...` after every single task. Use `xvfb-run` if a display is missing.
- **Reversion**: If a test fix takes >20 lines or touches out-of-scope files, revert the task and log the blocker.

## 🛑 Stopping Conditions
- Cease the session immediately if: the backlog is exhausted, tests fail unrecoverably, or a task requires breaking a public API / database schema.
