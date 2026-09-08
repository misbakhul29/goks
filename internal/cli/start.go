package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// StartCmd returns the `goks start` subcommand.
func StartCmd() *cobra.Command {
	var port string

	cmd := &cobra.Command{
		Use:   "start [port]",
		Short: "Start the GoKS production server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				port = args[0]
			}

			cwd, _ := os.Getwd()
			serverPath := filepath.Join(cwd, ".goks", "build", "server")
			
			if _, err := os.Stat(serverPath); os.IsNotExist(err) {
				return fmt.Errorf("production build not found. Run 'goks build' first")
			}

			fmt.Println(color.CyanString("\n  🚀 Starting GoKS Production Server"))

			runCmd := exec.Command(serverPath)
			runCmd.Dir = cwd
			runCmd.Stdout = os.Stdout
			runCmd.Stderr = os.Stderr
			
			env := os.Environ()
			if port != "" {
				env = append(env, "GOKS_CHILD_PORT="+port)
			}
			runCmd.Env = env

			return runCmd.Run()
		},
	}

	cmd.Flags().StringVarP(&port, "port", "p", "", "Port to listen on (default 3000)")
	return cmd
}
