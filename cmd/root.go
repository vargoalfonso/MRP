package cmd

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var rootCmd = &cobra.Command{
	Use:   "go-template",
	Short: "Go Template — production-ready API service",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(HttpCmd)

	if err := godotenv.Load(); err != nil {
		if os.Getenv("APP_ENV") == "development" || os.Getenv("APP_ENV") == "" {
			slog.Warn("could not load .env file, falling back to system environment", slog.Any("error", err))
		}
	}
}

// Execute is the entry point called from main.
// It auto-selects the "http" sub-command when none is provided so the binary
// can be started simply as ./go-template without arguments.
func Execute() error {
	cmd, _, err := rootCmd.Find(os.Args[1:])
	if err == nil && cmd.Use == rootCmd.Use && cmd.Flags().Parse(os.Args[1:]) != pflag.ErrHelp {
		args := append([]string{"http"}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}
	return rootCmd.Execute()
}
