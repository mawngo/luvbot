package cmd

import (
	"fmt"
	"github.com/phsym/console-slog"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
	"luvbot/cmd/ig"
	"luvbot/cmd/profile"
	"os"
	"time"
)

func initLogLevel() *slog.LevelVar {
	level := &slog.LevelVar{}
	logger := slog.New(
		console.NewHandler(os.Stderr, &console.HandlerOptions{
			Level:      level,
			TimeFormat: time.Kitchen,
		}))
	slog.SetDefault(logger)
	cobra.EnableCommandSorting = false
	return level
}

type CLI struct {
	command *cobra.Command
}

// NewCLI create new CLI instance and set up application config.
func NewCLI() *CLI {
	level := initLogLevel()
	command := cobra.Command{
		Use:   "luvbot",
		Short: "Automatically liking Instagram posts",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			if lo.Must(cmd.Flags().GetBool("debug")) {
				level.Set(slog.LevelDebug)
			}
		},
	}
	command.PersistentFlags().Bool("debug", false, "Enable debug mode")
	command.AddCommand(
		profile.NewCmd(),
		ig.NewCmd(),
	)
	return &CLI{&command}
}

func (cli *CLI) Execute() {
	if err := cli.command.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}
