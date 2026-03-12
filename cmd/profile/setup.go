package profile

import (
	"fmt"
	"github.com/spf13/cobra"
	"log/slog"
	"luvbot/internal/browser"
	"luvbot/internal/config"
)

func newSetupCmd() *cobra.Command {
	f := browser.Flags{
		Headless: false,
		UserMode: true,
		Profile:  config.DefaultProfile,
	}

	command := cobra.Command{
		Use:   "setup",
		Short: "Open browser in headful mode for account setup",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			slog.Info("Opening browser in headful mode for account setup", slog.String("profile", f.Profile))
			p, err := browser.NewPage(f)
			if err != nil {
				slog.Error("Cannot open browser", slog.Any("err", err))
			}
			defer p.Close()
			slog.Info("Press Enter to close the browser")
			_, _ = fmt.Scanln()
		},
	}
	browser.BindCmdFlags(&command, &f)
	return &command
}
