package browser

import (
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"log/slog"
	"luvbot/internal/config"
	"path/filepath"
	"runtime/debug"
	"time"
)

type Flags struct {
	Profile string
	XVFB    []string

	Headless bool
	UserMode bool
	Leakless bool
	Stealth  bool
}

func NewHeadlessFlags() Flags {
	return Flags{
		Headless: true,
		UserMode: true,
		Profile:  config.DefaultProfile,
	}
}

func BindCmdFlags(cmd *cobra.Command, flags *Flags) {
	cmd.Flags().BoolVar(&flags.Headless, "headless", flags.Headless, "enable headless mode")
	cmd.Flags().BoolVar(&flags.UserMode, "usermode", flags.UserMode, "enable usermode")
	cmd.Flags().BoolVar(&flags.Leakless, "leakless", flags.Leakless, "enable leakless")
	cmd.Flags().BoolVar(&flags.Stealth, "stealth", flags.Stealth, "enable stealth mode")

	cmd.Flags().StringVar(&flags.Profile, "profile", flags.Profile, "select profile to use")
	cmd.Flags().StringSliceVar(&flags.XVFB, "xvfb", flags.XVFB, "enable XVFB mode")
}

func Execute(f Flags, handler func(page *Page) error) error {
	p, err := NewPage(f)
	if err != nil {
		slog.Error("Cannot open browser", slog.Any("err", err))
	}
	defer p.Close()
	defer p.RecoverWithScreenShot()
	err = handler(p)
	return err
}

func NewPage(f Flags) (*Page, error) {
	l := lo.Ternary(f.UserMode, launcher.NewUserMode(), launcher.New())
	isXVFBEnabled := len(f.XVFB) > 0 && f.XVFB[0] != "false"
	if isXVFBEnabled {
		f.Headless = false
		if f.XVFB[0] == "true" {
			f.XVFB = []string{"-a"}
		}
		l = l.XVFB(f.XVFB...)
	}
	l = l.UserDataDir(filepath.Join(config.ProfilesDirectory, lo.Ternary(f.Profile == "", config.DefaultProfile, f.Profile))).
		Leakless(f.Leakless).
		Headless(f.Headless)
	l.Set("disable-blink-features", "AutomationControlled")
	l.Set("disable-features", "CreateDesktopShortcut")
	l.Set("window-size", "1600,900")
	u, err := l.Launch()
	if err != nil {
		return nil, err
	}

	b := rod.New().ControlURL(u).NoDefaultDevice()
	if err := b.Connect(); err != nil {
		defer l.Kill()
		return nil, err
	}

	var p *rod.Page
	if f.Stealth {
		p, err = stealth.Page(b)
	} else {
		p, err = b.Page(proto.TargetCreateTarget{})
	}

	if err != nil {
		defer l.Kill()
		defer b.Close()
		return nil, err
	}
	return &Page{
		Page:     p,
		launcher: l,
		browser:  b,
	}, nil
}

type Page struct {
	*rod.Page
	launcher *launcher.Launcher
	browser  *rod.Browser
}

func (p *Page) Close() {
	defer p.launcher.Kill()
	defer func() {
		err := p.browser.Close()
		if err != nil {
			slog.Debug("Cannot close browser", slog.Any("err", err))
		}
	}()
	defer func() {
		err := p.Page.Close()
		if err != nil {
			slog.Debug("Cannot close page", slog.Any("err", err))
		}
	}()
}

// RecoverWithScreenShot catch panic and save a screenshot.
func (p *Page) RecoverWithScreenShot() {
	if r := recover(); r != nil {
		defer fmt.Printf("Stack trace:\n%s", debug.Stack())

		screenshot := filepath.Join(config.ErrorScreenshotsDirectory, time.Now().Format("2006-01-02-150405_panic.png"))
		p.MustScreenshot(screenshot)
		defer func() {
			if sr := recover(); sr != nil {
				slog.Error("Error",
					slog.Any("err", r),
					slog.String("screenshot", "error"),
					slog.Any("screenshotErr", sr))
			}
		}()
		slog.Error("Error", slog.Any("err", r), slog.String("screenshot", screenshot))
	}
}
