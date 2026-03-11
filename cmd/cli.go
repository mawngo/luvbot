package cmd

import (
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/mawngo/go-try/v2"
	"github.com/phsym/console-slog"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

var retryOptions = try.NewOptions(try.WithExponentialBackoff(2*time.Second, 10*time.Second), try.WithAttempts(5))

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
	flags := botFlags{
		UserMode:          true,
		Headless:          true,
		MaxScrollPosts:    10_000,
		MaxLikes:          1000,
		MaxContinuedLikes: 30,
		FistLoadTimeout:   30 * time.Second,
	}

	command := cobra.Command{
		Use:   "luvbot",
		Short: "Automatically liking Instagram posts",
		Args:  cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if lo.Must(cmd.PersistentFlags().GetBool("debug")) {
				level.Set(slog.LevelDebug)
			}
			if lo.Must(cmd.PersistentFlags().GetBool("login")) {
				// Allow longer wait time so that user can log in.
				flags.FistLoadTimeout = 10 * time.Minute
			}
			if lo.Must(cmd.PersistentFlags().GetBool("xvfb")) {
				flags.XVFB = true
				flags.Headless = false
			}
			return nil
		},
		Run: func(cmd *cobra.Command, _ []string) {
			start := time.Now()
			l := lo.Ternary(flags.UserMode, launcher.NewUserMode(), launcher.New())
			l = l.UserDataDir(filepath.Join("profiles", lo.Must(cmd.PersistentFlags().GetString("userdir")))).
				Leakless(flags.Leakless).
				Headless(flags.Headless)
			if flags.XVFB {
				l = l.XVFB("-a")
			}
			l.Set("disable-blink-features", "AutomationControlled")
			l.Set("disable-features", "CreateDesktopShortcut")
			l.Set("window-size", "1600,900")
			u := l.MustLaunch()
			defer l.Kill()

			b := rod.New().ControlURL(u).NoDefaultDevice()
			b.MustConnect()
			defer b.Close()

			p := b.MustPage()
			defer p.Close()
			defer func() {
				if r := recover(); r != nil {
					screenshot := filepath.Join("errors", time.Now().Format("2006-01-02-150405_panic.png"))
					p.MustScreenshot(screenshot)
					slog.Error("Error", slog.Any("err", r), slog.String("screenshot", screenshot))
				}
			}()
			p.MustNavigate("https://www.instagram.com/")
			slog.Info("Waiting for Instagram page to load...")
			p.Timeout(flags.FistLoadTimeout).MustElement("article:not([data-index]) div > div:last-child svg[aria-label$='Save']")

			likedCnt := 0
			alreadyLikedCnt := 0
			article := p.MustElement("article:not([data-index])")
			for i := range flags.MaxScrollPosts {
				if i > 0 {
					// Scroll to the next article.
					time.Sleep(time.Second*2 + time.Duration(rand.Intn(500))*time.Millisecond)
					var err error
					article, err = try.GetWithOptions(article.Next, retryOptions)
					if err != nil {
						slog.Error("Cannot scroll to next article", slog.Any("err", err))
						break
					}
				}
				article.MustScrollIntoView()
				article.MustEval(fmt.Sprintf(`() => this.setAttribute('data-index', '%d')`, i))
				if _, err := article.Element("div"); err != nil {
					// Some articles are just an empty element.
					continue
				}
				if _, err := article.ElementX("div/div//span[text()='Ad']"); err == nil {
					// Some articles are ads.
					slog.Info("Skip", slog.Int("i", i), slog.String("reason", "ad article"))
					continue
				}
				if _, err := article.ElementX(`div//span[text()="You're all caught up"]`); err == nil {
					if flags.ExtendedScroll {
						continue
					}
					slog.Info("Stopped", slog.String("reason", "you're all caught up"))
					break
				}
				if article.Object.Description != "article" {
					slog.Info("Skip", slog.Int("i", i), slog.String("reason", "not an article"))
					// Not an article.
					continue
				}

				// Wait for the article to be fully loaded.
				slog.Debug("Waiting for article to be fully loaded...")
				p.MustElement(fmt.Sprintf("article[data-index='%d'] div > div:last-child svg[aria-label$='Save']", i))
				slog.Debug("Article is fully loaded")

				meta, err := extractPostMetadata(article)
				if err != nil {
					slog.Error("Failed to extract post metadata", slog.Any("err", err))
					break
				}
				slog.Info("Post",
					slog.Int("i", i),
					slog.String("u", "@"+meta.Username),
					slog.String("at", meta.Time.Format("2006-01-02 15:05")),
					slog.Bool("followed", meta.Followed),
					slog.Bool("liked", meta.Liked),
				)

				// Handling like and limit.
				if meta.Followed {
					if meta.Liked {
						alreadyLikedCnt++
					} else {
						slog.Debug("Liking...")
						alreadyLikedCnt = 0
						meta.LikeBtn.MustClick()
						likedCnt++
						slog.Debug("Liked")
					}
				}

				if likedCnt > flags.MaxLikes {
					slog.Info("Stopped", slog.String("reason", "Reached maximum likes"))
					break
				}
				if alreadyLikedCnt >= flags.MaxContinuedLikes {
					slog.Info("Stopped", slog.String("reason", "It's likely that there is no more new post"))
					break
				}
			}
			slog.Info("Completed",
				slog.Int("liked", likedCnt),
				slog.String("took", time.Since(start).String()))
		},
	}

	command.Flags().SortFlags = false
	command.Flags().BoolVar(&flags.Headless, "headless", flags.Headless, "Enable headless mode")
	command.Flags().BoolVar(&flags.UserMode, "usermode", flags.UserMode, "Enable usermode")
	command.Flags().BoolVar(&flags.Leakless, "leakless", flags.Leakless, "Enable leakless")
	command.Flags().BoolVar(&flags.ExtendedScroll, "ext", flags.ExtendedScroll, "Keep scroll posts after reached 'All caught up'")

	command.PersistentFlags().Bool("debug", false, "Enable debug mode")
	command.PersistentFlags().String("userdir", "chrome-user-data-test", "Enable debug mode")
	command.PersistentFlags().Bool("login", false, "Enable login setup mode")
	command.PersistentFlags().Bool("xvfb", false, "Enable login xvfb mode")
	return &CLI{&command}
}

func (cli *CLI) Execute() {
	if err := cli.command.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}

type botFlags struct {
	Headless       bool
	UserMode       bool
	Leakless       bool
	ExtendedScroll bool
	XVFB           bool

	MaxScrollPosts    int
	MaxLikes          int
	MaxContinuedLikes int
	FistLoadTimeout   time.Duration
}

type PostMetadata struct {
	Username       string
	IsMultipleUser bool
	Time           time.Time
	Followed       bool
	Liked          bool
	LikeBtn        *rod.Element
}

func extractPostMetadata(article *rod.Element) (meta PostMetadata, err error) {
	headerEl, err := article.Element("div > div")
	if err != nil {
		return PostMetadata{}, errors.New("header group not found")
	}

	unameEls := headerEl.MustElements("span > div > a[href^='/'] span, span > a[href='#'] span")
	meta.IsMultipleUser = len(unameEls) > 1
	if !unameEls.Empty() {
		meta.Username, err = unameEls.First().Text()
		if err != nil {
			return PostMetadata{}, errors.New("cannot extract username text")
		}
	} else {
		return PostMetadata{}, errors.New("username element not found")
	}

	if postTimeEl, err := headerEl.Element("time"); err == nil {
		rawPostTime, err := postTimeEl.Attribute("datetime")
		if err != nil || rawPostTime == nil {
			return PostMetadata{}, errors.New("post time element missing datetime attribute")
		}

		meta.Time, err = time.Parse(time.RFC3339, *rawPostTime)
		if err != nil {
			return PostMetadata{}, fmt.Errorf("cannot parse post time: %s", *rawPostTime)
		}
	} else {
		return PostMetadata{}, errors.New("post time element not found")
	}

	if !meta.IsMultipleUser {
		if _, err := headerEl.ElementX("div//div[text()='Follow']"); err != nil {
			meta.Followed = true
		}
	}

	if meta.LikeBtn, err = article.Element("div > div:last-child section svg[aria-label$='ike']"); err == nil {
		label, err := meta.LikeBtn.Attribute("aria-label")
		if err != nil || label == nil {
			return PostMetadata{}, errors.New("like icon element missing aria-label attribute")
		}
		meta.Liked = *label != "Like"
	} else {
		return PostMetadata{}, errors.New("like icon element not found")
	}
	// Hover post content to remove the popup caused by hover over the username.
	// IDK why it happened.
	article.MustElement("div > div:nth-child(2)").MustHover()
	return meta, nil
}
