package cli

import (
	"os"

	"github.com/misbakhul29/goks/internal/lsp"
	"github.com/spf13/cobra"
)

// LSPCmd starts the Language Server Protocol server for GOX files.
func LSPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lsp",
		Short: "Start the GOX Language Server Protocol (LSP)",
		Long:  "Starts the Language Server Protocol server for .gox files over stdio (JSON-RPC 2.0). Connect from VS Code, Neovim, Helix, or any LSP-compatible editor.",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := lsp.NewServer()
			return server.Serve(os.Stdin, os.Stdout)
		},
	}
}
