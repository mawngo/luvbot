package igbot

import (
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mawngo/go-errors"
	"github.com/mawngo/go-try/v2"
	"github.com/spf13/cobra"
	"log/slog"
	"luvbot/internal/browser"
	"luvbot/internal/config"
	"math/rand"
	"time"
)

const HomePageURL = "https://www.instagram.com/"

func NewLikePostsFlags() LikePostFlags {
	return LikePostFlags{
		MaxScrollPosts:    5000,
		MaxLikes:          1000,
		MaxContinuedLikes: 10,
		FistLoadTimeout:   1 * time.Minute,
		ElementTimeout:    5 * time.Minute,
	}
}

func BindCmdLikePostsFlags(command *cobra.Command, f *LikePostFlags) {
	command.Flags().BoolVar(&f.ExtendedScroll, "ext", f.ExtendedScroll, "Keep scroll posts or stories after reached the end")
	command.Flags().BoolVar(&f.SeenOnly, "seen", f.SeenOnly, "Does not like posts")
	command.Flags().DurationVar(&f.FistLoadTimeout, "first-load-timeout", f.FistLoadTimeout, "Timeout waiting for the first load to show up")
	command.Flags().BoolVar(&f.EarlyStop, "early-stop", f.EarlyStop, "Stop liking posts after 3 continuous liking")
}

type LikePostFlags struct {
	ExtendedScroll    bool
	MaxScrollPosts    int
	MaxLikes          int
	MaxContinuedLikes int
	EarlyStop         bool
	SeenOnly          bool

	FistLoadTimeout time.Duration
	ElementTimeout  time.Duration
}

func prepareFlag(f LikePostFlags) LikePostFlags {
	if f.EarlyStop {
		f.MaxContinuedLikes = 3
	}
	return f
}

// LikePosts Open IG and like all posts in the feed.
func LikePosts(p *browser.Page, f LikePostFlags) (int, error) {
	f = prepareFlag(f)

	if info := p.MustInfo(); info.URL != HomePageURL {
		p.MustNavigate(HomePageURL)
		slog.Info("Waiting for Instagram page to load...")
		p.MustWaitLoad()
	}

	slog.Info("Waiting for first post...")
	if _, err := p.Timeout(f.FistLoadTimeout).Element("article:not([data-index]) div > div:last-child svg[aria-label$='Save']"); err != nil {
		if !f.ExtendedScroll && isPostsAllCatchUp(p.MustElement("article:not([data-index])")) {
			slog.Info("Stopped", slog.String("reason", "you're all caught up"))
			return 0, nil
		}
		return 0, err
	}

	likedCnt := 0
	alreadyLikedCnt := 0
	notFollowedCnt := 0
	article := p.MustElement("article:not([data-index])")
	for i := range f.MaxScrollPosts {
		if i > 0 {
			// Scroll to the next article.
			WaitBetweenPosts()
			var err error
			article, err = try.GetWithOptions(func() (*rod.Element, error) {
				next, err := article.Next()
				if err != nil {
					return article, err
				}
				return next, nil
			}, config.ElementRetryOpt)
			if err != nil {
				html, err := article.HTML()
				if err != nil {
					html = err.Error()
				}
				slog.Error("Cannot scroll to next article",
					slog.Any("err", err),
					slog.String("currentEl", html),
				)
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
		if isPostsAllCatchUp(article) {
			if f.ExtendedScroll {
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
		p.Timeout(f.ElementTimeout).MustElement(fmt.Sprintf("article[data-index='%d'] div > div:last-child svg[aria-label$='Save']", i))
		slog.Debug("Article is fully loaded")

		slog.Debug("Parsing metadata...")
		meta, err := extractPostMetadata(article)
		if err != nil {
			slog.Error("Failed to extract post metadata", slog.Any("err", err))
			p.MustErrorScreenshotForDebug("failed_metadata", meta.Username)
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
			notFollowedCnt = 0
			if meta.Liked {
				alreadyLikedCnt++
			} else {
				slog.Debug("Liking...")
				alreadyLikedCnt = 0
				if !f.SeenOnly {
					if err := meta.LikeBtn.Timeout(2*time.Second).Click(proto.InputMouseButtonLeft, 2); err != nil {
						slog.Warn("Failed to click", slog.Any("err", err),
							slog.String("u", meta.Username),
							slog.String("btn", "like"))
						p.MustErrorScreenshotForDebug("failed_like", meta.Username)
					}
				}
				likedCnt++
				slog.Debug("Liked")
			}
		} else {
			notFollowedCnt++
		}

		if likedCnt > f.MaxLikes {
			slog.Info("Stopped", slog.String("reason", "Reached maximum likes"))
			break
		}
		if alreadyLikedCnt >= f.MaxContinuedLikes || notFollowedCnt >= f.MaxContinuedLikes {
			slog.Info("Stopped", slog.String("reason", "It's likely that there is no more new post"))
			break
		}
	}
	return likedCnt, nil
}

func isPostsAllCatchUp(el *rod.Element) bool {
	if _, err := el.ElementX(`div//span[text()="You're all caught up"]`); err == nil {
		return true
	}
	if _, err := el.ElementX(`div//span[text()="You've completely caught up"]`); err == nil {
		return true
	}
	return false
}

func WaitBetweenPosts() {
	time.Sleep(2*time.Second + time.Duration(rand.Intn(500))*time.Millisecond)
}

type postMetadata struct {
	Username       string
	IsMultipleUser bool
	Time           time.Time
	Followed       bool
	Liked          bool
	LikeBtn        *rod.Element
}

func extractPostMetadata(article *rod.Element) (meta postMetadata, err error) {
	headerEl, err := article.Element("div > div")
	if err != nil {
		return postMetadata{}, errors.Newf("header group not found")
	}

	unameEls := headerEl.MustElements(`span > div > a[href^="/"]:not([href^="/reels/"]):not([href^="/explore/"]) span:not(:has(>*)), span > a[href="#"] span`)
	meta.IsMultipleUser = len(unameEls) > 1
	if !unameEls.Empty() {
		meta.Username, err = unameEls.First().Text()
		if err != nil {
			return postMetadata{}, errors.Newf("cannot extract username text")
		}
	} else {
		return postMetadata{}, errors.Newf("username element not found")
	}

	if postTimeEl, err := headerEl.Element("time"); err == nil {
		rawPostTime, err := postTimeEl.Attribute("datetime")
		if err != nil || rawPostTime == nil {
			return postMetadata{}, errors.Newf("post time element missing datetime attribute")
		}

		meta.Time, err = time.Parse(time.RFC3339, *rawPostTime)
		if err != nil {
			return postMetadata{}, errors.Newf("cannot parse post time: %s", *rawPostTime)
		}
	} else {
		return postMetadata{}, errors.Newf("post time element not found")
	}

	if !meta.IsMultipleUser {
		if _, err := headerEl.ElementX("div//div[text()='Follow']"); err != nil {
			meta.Followed = true
		}
	}

	if meta.LikeBtn, err = article.Element("div > div:last-child section svg[aria-label$='ike']"); err == nil {
		label, err := meta.LikeBtn.Attribute("aria-label")
		if err != nil || label == nil {
			return postMetadata{}, errors.Newf("like icon element missing aria-label attribute")
		}
		meta.Liked = *label != "Like"
	} else {
		return postMetadata{}, errors.Newf("like icon element not found")
	}
	// Hover post content to remove the popup caused by hover over the username.
	// IDK why it happened.
	article.MustElement("div > div:nth-child(2)").MustHover()
	return meta, nil
}
