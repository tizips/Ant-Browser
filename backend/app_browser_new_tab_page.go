package backend

import (
	"strings"
	"time"

	"ant-chrome/backend/internal/logger"
)

var (
	browserNewTabRedirectInitialGrace = 2 * time.Second
	browserNewTabRedirectPollInterval = 500 * time.Millisecond
)

func (a *App) browserNewTabRedirectURL(profile *BrowserProfile, launchedAt time.Time) (string, error) {
	return a.browserStartPageURL(profile, launchedAt)
}

func isBrowserBlankNewTabURL(rawURL string) bool {
	value := strings.TrimSpace(strings.ToLower(rawURL))
	value = strings.TrimRight(value, "/")
	switch value {
	case "", "about:blank", "chrome://newtab", "chrome://new-tab-page", "chrome://new-tab-page-third-party":
		return true
	default:
		return false
	}
}

func (a *App) watchBrowserNewTabRedirects(profile *BrowserProfile, debugPort int) {
	if a == nil || a.browserMgr == nil || profile == nil || debugPort <= 0 {
		return
	}
	if browserNewTabRedirectInitialGrace > 0 {
		time.Sleep(browserNewTabRedirectInitialGrace)
	}

	redirectURL, err := a.browserNewTabRedirectURL(profile, time.Now())
	if err != nil {
		logger.New("Browser").Warn("默认新标签页生成失败",
			logger.F("profile_id", profile.ProfileId),
			logger.F("error", err.Error()),
		)
		return
	}
	if strings.TrimSpace(redirectURL) == "" {
		return
	}

	log := logger.New("Browser")
	redirected := map[string]struct{}{}
	for {
		if !a.isProfileRunningOnDebugPort(profile.ProfileId, debugPort) {
			return
		}

		targets, err := fetchBrowserDebugTargets(debugPort)
		if err != nil {
			time.Sleep(browserNewTabRedirectPollInterval)
			continue
		}
		for _, target := range targets {
			if !strings.EqualFold(strings.TrimSpace(target.Type), "page") || strings.TrimSpace(target.WebSocketDebuggerUrl) == "" {
				continue
			}
			targetID := strings.TrimSpace(target.ID)
			if targetID == "" {
				targetID = strings.TrimSpace(target.WebSocketDebuggerUrl)
			}
			if _, ok := redirected[targetID]; ok {
				continue
			}
			if !isBrowserBlankNewTabURL(target.URL) {
				continue
			}
			if err := cdpCallWebSocket(target.WebSocketDebuggerUrl, "Page.navigate", map[string]any{"url": redirectURL}, 2*time.Second); err != nil {
				continue
			}
			redirected[targetID] = struct{}{}
			log.Info("空标签页已跳转到默认页",
				logger.F("profile_id", profile.ProfileId),
				logger.F("debug_port", debugPort),
				logger.F("target_id", targetID),
			)
		}
		time.Sleep(browserNewTabRedirectPollInterval)
	}
}
