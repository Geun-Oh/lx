package cmd

import (
	"fmt"
	"os"

	"github.com/Geun-Oh/lx/internal/core"
	"github.com/spf13/cobra"
)

var (
  keyword string
  rootCmd = &cobra.Command{
  	Use:   "lx",
  	Short: "lx is a tool for running commands and filtering their output",
  	Long: `lx is a tool for running commands and filtering their output.
  It is similar to the 'docker logs' command, but with additional filtering capabilities(WIP).`,
  	Run: func(cmd *cobra.Command, args []string) {
  		core.Extract(keyword, args)
	  },
  }
)

func init() {
    cobra.OnInitialize()

    rootCmd.PersistentFlags().StringVarP(&keyword, "keyword", "k", "", "keyword that you want to filter")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
