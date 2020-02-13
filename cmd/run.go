package cmd

import (
	"fmt"
	"log"

	"github.com/pgm/goconseq/run"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var (
	stateDir string

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("run called")
			stats, err := run.RunRulesInFile(stateDir, args[0])
			if err != nil {
				log.Fatalf("%s", err)
			}
			log.Printf("Executions: %d, ExistingAppliedRules: %d", stats.Executions, stats.ExistingAppliedRules)
		},
	}
)

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().StringVarP(&stateDir, "dir", "", "state", "Directory to store working results (defaults to 'state')")
}
