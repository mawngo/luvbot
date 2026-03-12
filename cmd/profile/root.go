package profile

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	command := cobra.Command{
		Use:     "profile",
		Aliases: []string{"acc"},
		Short:   "Profile account management",
	}
	command.AddCommand(newSetupCmd())
	return &command
}
