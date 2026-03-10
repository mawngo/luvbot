package main

import (
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"math/rand"
	"path/filepath"
	"time"
)

const MaxScrollPosts = 100_000
const MaxLikes = 1000
const MaxContinuedLikes = 20

func main() {
	l := launcher.NewUserMode().UserDataDir(filepath.Join("profiles", "chrome-user-data-test")).Leakless(false)
	l.Set("disable-features", "CreateDesktopShortcut")
	u := l.MustLaunch()
	defer l.Kill()

	b := rod.New().ControlURL(u).NoDefaultDevice()
	b.MustConnect()
	defer b.Close()

	p := b.MustPage()
	defer p.Close()

	p.MustNavigate("https://www.instagram.com/")
	p.MustElement("article:not([id]) div > div:last-child svg[aria-label$='ike']")

	likedCnt := 0
	alreadyLikedCnt := 0

	article := p.MustElement("article:not([data-index])")
	for i := range MaxScrollPosts {
		if i > 0 {
			// Scrolling.
			var err error
			article, err = article.Next()
			if err != nil {
				fmt.Printf("Error scroll next article: %s\n", err.Error())
				break
			}
		}
		article.MustScrollIntoView()
		time.Sleep(time.Second*2 + time.Duration(rand.Intn(500))*time.Millisecond)
		article.MustEval(fmt.Sprintf(`() => this.setAttribute('data-index', '%d')`, i))
		if _, err := article.Element("div"); err != nil {
			// Some articles are just an empty element.
			continue
		}
		if _, err := article.ElementX("div/div//span[text()='Ad']"); err == nil {
			// Some articles are ads.
			fmt.Println("Ad - Skipped")
			continue
		}
		// Wait for the article to be fully loaded.
		p.MustElement(fmt.Sprintf("article[data-index='%d'] div > div:last-child svg[aria-label$='ike']", i))

		meta, err := extractPostMetadata(article)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			break
		}
		meta.Println()
		if meta.Followed {
			if meta.Liked {
				alreadyLikedCnt++
			} else {
				alreadyLikedCnt = 0
				meta.LikeBtn.MustClick()
				likedCnt++
			}
		}

		if likedCnt > MaxLikes {
			fmt.Println("Stopped. Reached maximum likes.")
			break
		}
		if alreadyLikedCnt >= MaxContinuedLikes {
			fmt.Println("Stopped. It's likely that there is no more new post.")
			break
		}
	}
	fmt.Println(likedCnt, "posts liked")
	fmt.Println("Completed.")
}

type PostMetadata struct {
	Username string
	Time     time.Time
	Followed bool
	Liked    bool
	LikeBtn  *rod.Element
}

func (m PostMetadata) Println() {
	fmt.Printf("@%s - %s", m.Username, m.Time.Format("2006-01-02 15:05"))
	if m.Liked {
		fmt.Print(" [Liked]")
	}
	if !m.Followed {
		fmt.Print(" [Not Followed]")
	}
	fmt.Println()
}

func extractPostMetadata(article *rod.Element) (meta PostMetadata, err error) {
	headerEl, err := article.Element("div > div")
	if err != nil {
		return PostMetadata{}, errors.New("header group not found")
	}

	if unameEl, err := headerEl.Element("span > div > a[href^='/'] > div span"); err == nil {
		meta.Username, err = unameEl.Text()
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

	if _, err := headerEl.ElementX("div//div[text()='Follow']"); err != nil {
		meta.Followed = true
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
	return meta, nil
}
