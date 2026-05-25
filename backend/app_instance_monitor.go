package backend

import (
	"ant-chrome/backend/internal/logger"
	"fmt"
	"os/exec"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	browserTabMonitorInitialGrace = 2 * time.Second
	browserTabMonitorPollInterval = 500 * time.Millisecond
	browserTabMonitorEmptySamples = 3
	browserTabMonitorCloseTimeout = 2 * time.Second
)

func (a *App) waitBrowserProcess(profileId string, monitor *browserProcessMonitor) {
	err := monitor.Wait()

	log := logger.New("Browser")
	debugPort := 0
	profileName := profileId
	shouldMonitorDetached := false

	a.browserMgr.Mutex.Lock()
	profile, exists := a.browserMgr.Profiles[profileId]
	wasRunning := exists && profile.Running
	if exists {
		profileName = profile.ProfileName
		debugPort = profile.DebugPort
	}
	a.browserMgr.Mutex.Unlock()

	if wasRunning && debugPort > 0 {
		snapshot, changed := a.waitForBrowserDebugReady(profileId, debugPort, browserLauncherDetachGraceWindow)
		if snapshot != nil && changed {
			log.Info("浏览器启动器进程退出后，调试接口延迟就绪",
				logger.F("profile_id", profileId),
				logger.F("debug_port", debugPort),
			)
			a.emitBrowserInstanceUpdated(snapshot)
		}

		a.browserMgr.Mutex.Lock()
		profile, exists = a.browserMgr.Profiles[profileId]
		if exists && profile.Running && profile.DebugPort == debugPort && profile.DebugReady && canConnectDebugPort(debugPort, 250*time.Millisecond) {
			delete(a.browserMgr.BrowserProcesses, profileId)
			profile.Pid = 0
			shouldMonitorDetached = true
		}
		a.browserMgr.Mutex.Unlock()
		if shouldMonitorDetached {
			log.Info("浏览器启动器进程已退出，切换为调试端口存活监控",
				logger.F("profile_id", profileId),
				logger.F("profile_name", profileName),
				logger.F("debug_port", debugPort),
			)
			a.waitDetachedBrowser(profileId, debugPort)
			return
		}
	}

	a.browserMgr.Mutex.Lock()
	profile, exists = a.browserMgr.Profiles[profileId]
	wasRunning = exists && profile.Running
	if exists {
		profileName = profile.ProfileName
		a.markProfileStoppedLocked(profileId, profile)
	}
	a.browserMgr.Mutex.Unlock()

	if a.ctx == nil {
		return
	}

	if wasRunning && err != nil {
		if exists && profile != nil {
			profile.LastError = fmt.Sprintf("实例运行异常退出：%s", err.Error())
		}
		log.Error("浏览器进程异常退出", logger.F("profile_id", profileId), logger.F("profile_name", profileName), logger.F("error", err))
		runtime.EventsEmit(a.ctx, "browser:instance:crashed", map[string]interface{}{
			"profileId":   profileId,
			"profileName": profileName,
			"error":       err.Error(),
		})
	} else {
		runtime.EventsEmit(a.ctx, "browser:instance:stopped", profileId)
	}
}

func (a *App) watchBrowserTabs(profileId string, debugPort int) {
	if a == nil || a.browserMgr == nil || debugPort <= 0 {
		return
	}
	if browserTabMonitorInitialGrace > 0 {
		time.Sleep(browserTabMonitorInitialGrace)
	}

	log := logger.New("Browser")
	emptySamples := 0
	for {
		if !a.isProfileRunningOnDebugPort(profileId, debugPort) {
			return
		}

		count, err := browserDebugPageTargetCount(debugPort, browserDebugProbeTimeout)
		if err != nil {
			emptySamples = 0
			time.Sleep(browserTabMonitorPollInterval)
			continue
		}
		if count > 0 {
			emptySamples = 0
			time.Sleep(browserTabMonitorPollInterval)
			continue
		}

		emptySamples++
		if emptySamples < browserTabMonitorEmptySamples {
			time.Sleep(browserTabMonitorPollInterval)
			continue
		}

		cmd, profileName, ok := a.browserProcessForAutoClose(profileId, debugPort)
		if !ok {
			return
		}
		if !tryCloseBrowserViaCDP(debugPort, browserTabMonitorCloseTimeout) && cmd != nil && cmd.Process != nil {
			_ = a.stopBrowserProcess(cmd)
		}

		a.browserMgr.Mutex.Lock()
		profile, exists := a.browserMgr.Profiles[profileId]
		if !exists || profile == nil || !profile.Running || profile.DebugPort != debugPort {
			a.browserMgr.Mutex.Unlock()
			return
		}
		a.markProfileStoppedLocked(profileId, profile)
		a.browserMgr.Mutex.Unlock()

		log.Info("检测到所有标签页已关闭，实例已停止",
			logger.F("profile_id", profileId),
			logger.F("profile_name", profileName),
			logger.F("debug_port", debugPort),
		)
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "browser:instance:stopped", profileId)
		}
		return
	}
}

func (a *App) isProfileRunningOnDebugPort(profileId string, debugPort int) bool {
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()
	profile, exists := a.browserMgr.Profiles[profileId]
	return exists && profile != nil && profile.Running && profile.DebugPort == debugPort
}

func (a *App) browserProcessForAutoClose(profileId string, debugPort int) (*exec.Cmd, string, bool) {
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()
	profile, exists := a.browserMgr.Profiles[profileId]
	if !exists || profile == nil || !profile.Running || profile.DebugPort != debugPort {
		return nil, "", false
	}
	return a.browserMgr.BrowserProcesses[profileId], profile.ProfileName, true
}

func (a *App) waitDetachedBrowser(profileId string, debugPort int) {
	const (
		pollInterval = 500 * time.Millisecond
		maxMisses    = 3
	)

	log := logger.New("Browser")
	misses := 0
	for {
		if canConnectDebugPort(debugPort, 250*time.Millisecond) {
			misses = 0
			time.Sleep(pollInterval)
			continue
		}

		misses++
		if misses < maxMisses {
			time.Sleep(pollInterval)
			continue
		}

		profileName := profileId
		a.browserMgr.Mutex.Lock()
		profile, exists := a.browserMgr.Profiles[profileId]
		if !exists || !profile.Running || profile.DebugPort != debugPort {
			a.browserMgr.Mutex.Unlock()
			return
		}
		profileName = profile.ProfileName
		a.markProfileStoppedLocked(profileId, profile)
		a.browserMgr.Mutex.Unlock()

		log.Info("检测到浏览器调试端口关闭，实例已停止",
			logger.F("profile_id", profileId),
			logger.F("profile_name", profileName),
			logger.F("debug_port", debugPort),
		)
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "browser:instance:stopped", profileId)
		}
		return
	}
}
