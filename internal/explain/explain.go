// Package explain asks an optional language model for a plain-English, second
// opinion on a command. It is never on the hot path: the deterministic rules
// are what guard your shell. `oops-guard explain` is for the moment you've
// pasted a gnarly one-liner and want a human-readable account of what it does
// before you trust it.
package explain

import (
	"context"
	"strings"

	"github.com/agenticraptor/oops-guard/internal/analyzer"
	"github.com/agenticraptor/oops-guard/internal/llm"
)

const systemPrompt = `You are a careful shell-safety reviewer. Given a single command line, explain in plain English:
1. What the command actually does, step by step (briefly).
2. What — specifically — could be lost or broken if it is wrong, and whether that is reversible.
3. A one-line verdict: is this safe to run, run with care, or genuinely dangerous?

Be concise and concrete. Do not invent flags or behavior. If the command is harmless, say so plainly. Never include a preamble like "Sure" — start directly with the explanation. Output plain text, no markdown headers.`

// Explain returns the model's prose explanation of the command. The
// deterministic findings are passed along as hints so the model can build on
// them rather than starting cold.
func Explain(ctx context.Context, client llm.Client, command string, a analyzer.Assessment) (string, error) {
	var b strings.Builder
	b.WriteString("Command:\n")
	b.WriteString(command)
	b.WriteString("\n\n")
	if len(a.Findings) > 0 {
		b.WriteString("A static rule checker already flagged:\n")
		for _, f := range a.Findings {
			b.WriteString("- [")
			b.WriteString(f.Severity.Label())
			b.WriteString("] ")
			b.WriteString(f.Title)
			b.WriteString(": ")
			b.WriteString(f.Detail)
			b.WriteString("\n")
		}
		b.WriteString("\nConfirm, correct, or add to this, then give the verdict.")
	} else {
		b.WriteString("A static rule checker found nothing dangerous. Sanity-check that and give the verdict.")
	}

	return client.Complete(ctx, llm.Request{
		System:      systemPrompt,
		Prompt:      b.String(),
		MaxTokens:   600,
		Temperature: 0,
	})
}
