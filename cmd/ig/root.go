package ig

import (
	"github.com/spf13/cobra"
	"log/slog"
	"luvbot/internal/browser"
	"luvbot/internal/igbot"
	"time"
)

func NewCmd() *cobra.Command {
	f := flags{
		Flags:         browser.NewHeadlessFlags(),
		LikePostFlags: igbot.NewLikePostsFlags(),
	}

	command := cobra.Command{
		Use:   "ig",
		Short: "Automatically liking Instagram posts and stories",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			start := time.Now()
			_ = browser.Execute(f.Flags, func(p *browser.Page) error {
				postLikedCnt, err := igbot.LikePosts(p, f.LikePostFlags)
				if err != nil {
					slog.Error("Error liking posts", slog.Any("err", err))
					return nil
				}
				storiesLikedCnt, err := igbot.LikeStories(p, f.LikePostFlags)
				if err != nil {
					slog.Error("Error liking stories", slog.Any("err", err))
					return nil
				}
				slog.Info("Completed",
					slog.Int("posts", postLikedCnt),
					slog.Int("stories", storiesLikedCnt),
					slog.String("took", time.Since(start).String()))
				return nil
			})
		},
	}
	browser.BindCmdFlags(&command, &f.Flags)
	igbot.BindCmdLikePostsFlags(&command, &f.LikePostFlags)

	command.AddCommand(NewPostsCmd())
	command.AddCommand(NewStoriesCmd())
	return &command
}
