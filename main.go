// Command goks is the GoKS CLI entry point.
package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/misbakhul29/goks/internal/cli"
)

var banner = `
  ██████╗  ██████╗ ██╗  ██╗███████╗
 ██╔════╝ ██╔═══██╗██║ ██╔╝██╔════╝
 ██║  ███╗██║   ██║█████╔╝ ███████╗
 ██║   ██║██║   ██║██╔═██╗ ╚════██║
 ╚██████╔╝╚██████╔╝██║  ██╗███████║
  ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚══════╝
`

func main() {
	root := &cobra.Command{
		Use:   "goks",
		Short: "GoKS — Fullstack Go Framework",
		Long:  color.CyanString(banner) + "\n  " + color.WhiteString("The fullstack Go framework. Backend + WASM frontend, one language."),
		RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}

	root.AddCommand(
		cli.NewCmd(),
		cli.DevCmd(),
		cli.BuildCmd(),
		cli.StartCmd(),
		cli.GenerateCmd(),
		cli.PageCmd(),
		versionCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the GoKS version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(color.CyanString("GoKS") + " v0.1.0")
		},
	}
}
