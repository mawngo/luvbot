package ig

import (
	"github.com/spf13/cobra"
	"log/slog"
	"luvbot/internal/browser"
	"luvbot/internal/igbot"
	"time"
)

func NewStoriesCmd() *cobra.Command {
	f := flags{
		Flags:         browser.NewHeadlessFlags(),
		LikePostFlags: igbot.NewLikePostsFlags(),
	}

	command := cobra.Command{
		Use:   "stories",
		Short: "Automatically liking Instagram stories",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			start := time.Now()
			return browser.Execute(f.Flags, func(p *browser.Page) error {
				likedCnt, err := igbot.LikeStories(p, f.LikePostFlags)
				if err != nil {
					return err
				}
				slog.Info("Completed",
					slog.Int("liked", likedCnt),
					slog.String("took", time.Since(start).String()))
				return nil
			})
		},
	}
	browser.BindCmdFlags(&command, &f.Flags)
	igbot.BindCmdLikePostsFlags(&command, &f.LikePostFlags)
	return &command
}
