// Command oops-guard catches a genuinely destructive shell command before it
// runs — telling you exactly what would be lost — and asks you to confirm.
package main

import (
	"os"

	"github.com/agenticraptor/oops-guard/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
