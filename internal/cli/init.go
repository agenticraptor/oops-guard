package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/oops-guard/internal/shell"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "init <bash|zsh|fish>",
		Short:     "Print the shell integration snippet",
		ValidArgs: shell.Supported(),
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Long: `init prints the shell hook that makes oops-guard inspect commands before
they run. Add it to your shell startup file:

  bash   eval "$(oops-guard init bash)"      # ~/.bashrc
  zsh    eval "$(oops-guard init zsh)"       # ~/.zshrc
  fish   oops-guard init fish | source       # ~/.config/fish/config.fish`,
		RunE: func(cmd *cobra.Command, args []string) error {
			snippet, err := shell.Snippet(args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(os.Stdout, snippet)
			return nil
		},
	}
	return cmd
}
