package cmd

import (
	"fmt"
	"os"

	"github.com/Geun-Oh/lx/internal/core"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lx",
	Short: "lx is a tool for running commands and filtering their output",
	Long: `lx is a tool for running commands and filtering their output.
It is similar to the 'docker logs' command, but with additional filtering capabilities(WIP).`,
	Run: func(cmd *cobra.Command, args []string) {
		core.Extract(args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
