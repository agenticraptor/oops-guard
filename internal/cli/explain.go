package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/oops-guard/internal/analyzer"
	"github.com/agenticraptor/oops-guard/internal/config"
	"github.com/agenticraptor/oops-guard/internal/explain"
	"github.com/agenticraptor/oops-guard/internal/llm"
	"github.com/agenticraptor/oops-guard/internal/render"
)

func newExplainCmd() *cobra.Command {
	var (
		provider string
		model    string
		noColor  bool
	)
	cmd := &cobra.Command{
		Use:   "explain [command...]",
		Short: "Explain in plain English what a command does and what it risks",
		Long: `explain shows the deterministic rule findings and then — if a model is
available — asks it for a plain-English account of what the command does and
what could go wrong. This is the only command that ever uses a model, and it is
strictly opt-in: it never runs as part of the shell guard.

Provide a model by setting ANTHROPIC_API_KEY or OPENAI_API_KEY, or by running a
local Ollama. Your command text (not your files) is sent to the chosen model.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			line := strings.Join(args, " ")
			color := useColor(noColor, os.Stdout)
			a := analyzer.Analyze(line, analyzer.Options{})
			if err := render.Render(os.Stdout, a, render.Options{Format: render.FormatTerm, Color: color}); err != nil {
				return err
			}

			client, err := makeClient(provider, model)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nNo model available (%v).\n"+
					"Set ANTHROPIC_API_KEY or OPENAI_API_KEY, run Ollama, or see `oops-guard config`.\n", err)
				return nil
			}
			fmt.Printf("\n· asking %s (%s) …\n\n", client.Name(), client.Model())
			text, err := explain.Explain(cmd.Context(), client, line, a)
			if err != nil {
				return fmt.Errorf("model request failed: %w", err)
			}
			fmt.Println(text)
			return nil
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "model provider: anthropic, openai, or ollama (default: auto-detect)")
	cmd.Flags().StringVar(&model, "model", "", "model name (default: the provider's default)")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	cmd.Flags().SetInterspersed(false)
	return cmd
}

// makeClient builds an LLM client from the flags, the config file, and finally
// environment auto-detection.
func makeClient(provider, model string) (llm.Client, error) {
	cfg, _ := config.Load()
	if provider == "" {
		provider = cfg.AI.Provider
	}
	if model == "" {
		model = cfg.AI.Model
	}
	if provider == "" {
		return llm.Detect()
	}
	return llm.New(provider, model, "")
}
