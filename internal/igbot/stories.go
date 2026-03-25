package igbot

import (
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"log/slog"
	"luvbot/internal/browser"
	"time"
)

func LikeStories(p *browser.Page, f LikePostFlags) (int, error) {
	p.MustNavigate("https://www.instagram.com/")
	slog.Info("Waiting for Instagram page to load...")
	p.MustWaitLoad()
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
				slog.Info("Stopped", slog.String("reason", "no more story next"))
				break
			}
			nextBtn.MustClick()
			waitBetweenArticle()
			nextBtn, _ = container.Element(`div > div > div > svg[aria-label="Next"]`)
		}

		article := container.MustElement("div > div > div[style]:not(:has(>a)) > div[class]")
		if _, err := article.ElementX("div//span[text()='Ad']"); err == nil {
			slog.Info("Skip", slog.Int("i", i), slog.String("reason", "not an article"))
			continue
		}

		slog.Debug("Parsing metadata...")
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
		if meta.Liked {
			alreadyLikedCnt++
		} else {
			slog.Debug("Liking...")
			alreadyLikedCnt = 0
			if !f.SeenOnly && meta.LikeBtn != nil {
				meta.LikeBtn.MustClick()
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
	return likedCnt, nil
}

func openStories(p *browser.Page, loadTimeout time.Duration) (container *rod.Element) {
	p.Timeout(loadTimeout).MustElement(`div[data-pagelet="story_tray"] ul > li:has(div[role="button"]) div[role="button"]`)
	for i := range 1000 {
		waitBetweenArticle()
		storiesEl := p.MustElements(`div[data-pagelet="story_tray"] ul > li:has(div[role="button"]) div[role="button"]`)
		if len(storiesEl) == 0 {
			panic("Cannot detect story tray!")
		}
		if len(storiesEl) < i {
			return nil
		}

		el := storiesEl[i]
		el.MustClick()

		slog.Debug("Waiting for story container to open...")
		container = p.Timeout(loadTimeout).MustElement(`section:has(svg[aria-label="Close"]):has(div > div > div > div > div > div > div > div > video)`)
		closeBtn := container.MustElement(`svg[aria-label="Close"]`)

		// Exclude LIVE.
		if _, err := container.ElementX(`div//span[text()="LIVE"]`); err == nil {
			closeBtn.MustClick()
			slog.Debug("Next", slog.String("reason", "LIVE story"))
			continue
		}

		// Must have a like button to make sure this is an actual story.
		article := container.MustElement("div > div > div[style]:not(:has(>a))")
		if _, err := article.Element(`svg[aria-label$="ike"]`); err != nil {
			closeBtn.MustClick()
			slog.Debug("Next", slog.String("reason", "No Like button"))
			continue
		}
		break
	}
	return p.MustElement(`section:has(svg[aria-label="Close"]):has(div > div > div > div > div > div > div > div > video)`)
}

func extractStoryMetadata(container *rod.Element) (meta storyMetadata, err error) {
	unameEl, err := container.Element(`div > a[href^="/"] > span > span`)
	if err != nil {
		return meta, errors.New("username element not found")
	}
	meta.Username, err = unameEl.Text()
	if err != nil {
		return meta, errors.New("cannot extract username text")
	}

	if timeEl, err := container.Element("time"); err == nil {
		rawPostTime, err := timeEl.Attribute("datetime")
		if err != nil || rawPostTime == nil {
			return meta, errors.New("story time element missing datetime attribute")
		}

		meta.Time, err = time.Parse(time.RFC3339, *rawPostTime)
		if err != nil {
			return meta, fmt.Errorf("cannot parse story time: %s", *rawPostTime)
		}
	} else {
		return meta, errors.New("story time element not found")
	}

	if meta.LikeBtn, err = container.Element(`div[role="button"] svg[aria-label$="ike"]`); err == nil {
		label, err := meta.LikeBtn.Attribute("aria-label")
		if err != nil || label == nil {
			return meta, errors.New("like icon element missing aria-label attribute")
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
