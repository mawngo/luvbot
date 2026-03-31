package igbot

import (
	"github.com/go-rod/rod"
	"github.com/mawngo/go-errors"
	"github.com/mawngo/go-try/v2"
	"log/slog"
	"luvbot/internal/browser"
	"luvbot/internal/config"
	"math/rand"
	"time"
)

const (
	storyUsernameSelector  = `div > a[href^="/"] > span > span`
	storyArticleSelector   = ":scope > div > div > div:not(:has(>a)):has(>div)"
	storyContainerSelector = `section:has(svg[aria-label="Close"]):has(div > div > div > div > div > div > div > div > video,div > div > div > div > div > div > div > div > img)`
	storyTrayStorySelector = `div[data-pagelet="story_tray"] ul > li:has(div[role="button"]) div[role="button"]`

	storyCloseBtnSelector = `svg[aria-label="Close"]`
	storyLikeBtnSelector  = `svg[aria-label$="ike"]`
)

// LikeStories Open IG, open story view and like all likable stories in the view.
// This function will close the story view after completed.
func LikeStories(p *browser.Page, f LikePostFlags) (int, error) {
	if f.EarlyStop {
		f.MaxContinuedLikes = 3
	}

	if info := p.MustInfo(); info.URL != HomePageURL {
		p.MustNavigate(HomePageURL)
		slog.Info("Waiting for Instagram page to load...")
		p.MustWaitLoad()
	}

	slog.Info("Waiting for first story...")
	container := openStories(p, f.FistLoadTimeout)
	if container == nil {
		slog.Info("Stopped", slog.String("reason", "empty stories"))
		return 0, nil
	}

	likedCnt := 0
	alreadyLikedCnt := 0
	nextBtn, _ := container.Element(`div > div > div > svg[aria-label="Next"]`)
	for i := range f.MaxScrollPosts {
		if i > 0 {
			if nextBtn == nil {
				// Try to detect the next button, again, in case the last detection failed for some reason.
				nextBtn, _ = container.Element(`div > div > div > svg[aria-label="Next"]`)
				if nextBtn != nil {
					slog.Warn("Restored next", slog.String("reason", "last detection failed"))
				}
			}

			if nextBtn == nil {
				slog.Info("Stopped", slog.String("reason", "no more story next"))
				break
			}
			nextBtn.MustClick()
			time.Sleep(1*time.Second + 500*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond)
			nextBtn, _ = container.Element(`div > div > div > svg[aria-label="Next"]`)
		}

		article := container.Timeout(f.ElementTimeout).MustElement(storyArticleSelector)
		if _, err := article.ElementX("div//span[text()='Ad']"); err == nil {
			slog.Info("Skip", slog.Int("i", i), slog.String("reason", "ad article"))
			// Skip all story, go straight to the next article.
			nextBtn, _ = article.Next()
			continue
		}

		slog.Debug("Parsing metadata...")
		article = article.CancelTimeout()
		meta, err := extractStoryMetadata(article)
		if err != nil {
			slog.Error("Failed to extract story metadata", slog.Any("err", err))
			break
		}

		slog.Info("Story",
			slog.Int("i", i),
			slog.String("u", "@"+meta.Username),
			slog.String("at", meta.Time.Format("2006-01-02 15:05")),
			slog.Bool("liked", meta.Liked),
		)

		// Handling like and limit.
		if meta.LikeBtn == nil {
			// There is no like btn, it is pointless to keep scrolling this story.
			slog.Info("Skip", slog.Int("i", i), slog.String("reason", "no like button"))
			nextBtn, _ = article.Next()
			continue
		}

		if meta.Liked {
			alreadyLikedCnt++
			// If the story was already liked, then maybe
			// the bot already handled this article.
			// Jump to the next article immediately.
			nextBtn, _ = article.Next()
		} else {
			slog.Debug("Liking...")
			alreadyLikedCnt = 0
			if !f.SeenOnly {
				meta.LikeBtn.Timeout(f.ElementTimeout).MustClick()
			}
			likedCnt++
			slog.Debug("Liked")
		}

		if likedCnt > f.MaxLikes {
			slog.Info("Stopped", slog.String("reason", "Reached maximum likes"))
			break
		}
		if alreadyLikedCnt >= f.MaxContinuedLikes {
			slog.Info("Stopped", slog.String("reason", "It's likely that there is no more new story"))
			break
		}
	}

	slog.Debug("Closing stories view...")
	// Close the story view.
	closeBtn, err := container.Timeout(f.ElementTimeout).Element(`svg[aria-label="Close"]`)
	if err != nil {
		return likedCnt, errors.Wrapf(err, "cannot find close button on story view")
	}
	closeBtn.MustClick()
	return likedCnt, nil
}

func openStories(p *browser.Page, loadTimeout time.Duration) *rod.Element {
	p.Timeout(loadTimeout).MustElement(storyTrayStorySelector).MustScrollIntoView()

	for i := range 1000 {
		WaitBetweenArticle()
		storiesEl := p.MustElements(storyTrayStorySelector)
		if len(storiesEl) == 0 {
			panic("Cannot detect story tray!")
		}
		if len(storiesEl) <= i {
			return nil
		}

		container, err := try.GetWithOptions(func() (*rod.Element, error) {
			slog.Debug("Waiting for story container to open...")
			el := storiesEl[i]
			el.MustClick()

			c, err := p.Timeout(loadTimeout).Element(storyContainerSelector)
			if err != nil {
				// Close the story view if possible.
				if closeBtn, err := el.Element(`section svg[aria-label="Close"]`); err == nil {
					closeBtn.MustClick()
				}
			}
			return c, nil
		}, config.ElementRetryOpt)
		if err != nil {
			panic(err)
		}

		closeBtn := container.MustElement(storyCloseBtnSelector)

		// Exclude LIVE.
		if _, err := container.ElementX(`div//span[text()="LIVE"]`); err == nil {
			slog.Debug("Next", slog.String("reason", "LIVE story"))
			closeBtn.MustClick()
			continue
		}

		// Must have a like button to make sure this is an actual story.
		article := container.Timeout(2 * time.Second).MustElement(storyArticleSelector)
		if _, err := article.Element(storyLikeBtnSelector); err != nil {
			uname := "nil"
			if unameEl, err := container.Element(storyUsernameSelector); err == nil {
				uname = unameEl.MustText()
			}
			slog.Debug("Next", slog.String("reason", "no Like button"), slog.String("u", uname))
			closeBtn.MustClick()
			continue
		}
		break
	}
	return p.MustElement(storyContainerSelector)
}

func extractStoryMetadata(container *rod.Element) (meta storyMetadata, err error) {
	unameEl, err := container.Element(storyUsernameSelector)
	if err != nil {
		return meta, errors.Newf("username element not found")
	}
	meta.Username, err = unameEl.Text()
	if err != nil {
		return meta, errors.Newf("cannot extract username text")
	}

	if timeEl, err := container.Element("time"); err == nil {
		rawPostTime, err := timeEl.Attribute("datetime")
		if err != nil || rawPostTime == nil {
			return meta, errors.Newf("story time element missing datetime attribute")
		}

		meta.Time, err = time.Parse(time.RFC3339, *rawPostTime)
		if err != nil {
			return meta, errors.Newf("cannot parse story time: %s", *rawPostTime)
		}
	} else {
		return meta, errors.Newf("story time element not found")
	}

	if meta.LikeBtn, err = container.Timeout(2 * time.Second).Element(storyLikeBtnSelector); err == nil {
		label, err := meta.LikeBtn.Attribute("aria-label")
		if err != nil || label == nil {
			return meta, errors.Newf("like icon element missing aria-label attribute")
		}
		meta.Liked = *label != "Like"
	}
	return meta, nil
}

type storyMetadata struct {
	Username string
	Time     time.Time
	Liked    bool
	LikeBtn  *rod.Element
}
