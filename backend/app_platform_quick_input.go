package backend

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"ant-chrome/backend/internal/logger"
)

type platformQuickInputPayload struct {
	PlatformName string `json:"platformName"`
	ProfileName  string `json:"profileName"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	TwoFASecret  string `json:"twoFaSecret"`
	TwoFACode    string `json:"twoFaCode"`
}

func shouldInjectPlatformQuickInput(profile *BrowserProfile) bool {
	if browserProfilePlatformURL(profile) == "" {
		return false
	}
	return strings.TrimSpace(profile.Username) != "" ||
		strings.TrimSpace(profile.Password) != "" ||
		strings.TrimSpace(profile.TwoFASecret) != ""
}

func renderPlatformQuickInputScript(profile *BrowserProfile) (string, error) {
	if profile == nil {
		return "", fmt.Errorf("profile is nil")
	}
	twoFASecret := normalizeTOTPSecret(profile.TwoFASecret)
	twoFACode := ""
	if twoFASecret != "" {
		if code, err := browserStartPageTOTPCode(twoFASecret, time.Now()); err == nil {
			twoFACode = code
		}
	}
	payload := platformQuickInputPayload{
		PlatformName: browserStartPagePlatformName(profile),
		ProfileName:  strings.TrimSpace(profile.ProfileName),
		Username:     strings.TrimSpace(profile.Username),
		Password:     strings.TrimSpace(profile.Password),
		TwoFASecret:  twoFASecret,
		TwoFACode:    twoFACode,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return `(function(){
  const data = ` + string(data) + `;
  const rootId = 'ant-platform-quick-input-root';
  if (window.top !== window || document.getElementById(rootId)) return;
  function isEditable(el){
    if (!el) return false;
    const tag = (el.tagName || '').toLowerCase();
    if (el.isContentEditable) return true;
    if (tag !== 'input' && tag !== 'textarea') return false;
    if (el.disabled || el.readOnly) return false;
    const type = (el.getAttribute('type') || 'text').toLowerCase();
    return !['button','submit','reset','checkbox','radio','file','hidden','image','range','color'].includes(type);
  }
  function visible(el){
    if (!el) return false;
    const box = el.getBoundingClientRect();
    const style = window.getComputedStyle(el);
    return box.width > 0 && box.height > 0 && style.visibility !== 'hidden' && style.display !== 'none';
  }
  function query(selectors){
    for (const selector of selectors) {
      const list = Array.from(document.querySelectorAll(selector));
      const hit = list.find(function(el){ return isEditable(el) && visible(el); });
      if (hit) return hit;
    }
    return null;
  }
  function targetFor(kind){
    const active = document.activeElement;
    if (isEditable(active) && visible(active)) return active;
    if (kind === 'password') return query(['input[type="password"]']);
    if (kind === 'totp') return query([
      'input[autocomplete*="one-time-code" i]',
      'input[name*="otp" i]','input[id*="otp" i]',
      'input[name*="totp" i]','input[id*="totp" i]',
      'input[name*="2fa" i]','input[id*="2fa" i]',
      'input[name*="code" i]','input[id*="code" i]',
      'input[inputmode="numeric"]','input[type="tel"]','input[type="text"]'
    ]);
    return query([
      'input[autocomplete="username"]',
      'input[type="email"]',
      'input[name*="email" i]','input[id*="email" i]',
      'input[name*="user" i]','input[id*="user" i]',
      'input[name*="account" i]','input[id*="account" i]',
      'input[type="text"]','input:not([type])'
    ]);
  }
  function setValue(el, value){
    if (!el || value === '') return false;
    el.focus();
    if (el.isContentEditable) {
      document.execCommand('selectAll', false, null);
      document.execCommand('insertText', false, value);
    } else {
      const proto = el.tagName.toLowerCase() === 'textarea' ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
      const setter = Object.getOwnPropertyDescriptor(proto, 'value').set;
      setter.call(el, value);
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
    }
    return true;
  }
  function toast(message){
    let el = document.getElementById('ant-platform-quick-input-toast');
    if (!el) {
      el = document.createElement('div');
      el.id = 'ant-platform-quick-input-toast';
      el.style.cssText = 'position:fixed;right:18px;bottom:78px;z-index:2147483647;padding:8px 10px;border-radius:6px;background:#111827;color:#fff;font:12px -apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;box-shadow:0 8px 20px rgba(0,0,0,.22)';
      document.documentElement.appendChild(el);
    }
    el.textContent = message;
    clearTimeout(el.__antTimer);
    el.__antTimer = setTimeout(function(){ el.remove(); }, 1400);
  }
  function base32Decode(value){
    const alphabet='ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
    const clean=(value||'').toUpperCase().replace(/[\s=-]/g,'');
    let bits='', out=[];
    for(let i=0;i<clean.length;i++){ const idx=alphabet.indexOf(clean[i]); if(idx<0) throw new Error('invalid base32'); bits += idx.toString(2).padStart(5,'0'); }
    for(let j=0;j+8<=bits.length;j+=8){ out.push(parseInt(bits.slice(j,j+8),2)); }
    return new Uint8Array(out);
  }
  async function currentTOTP(){
    if (!data.twoFaSecret) return data.twoFaCode || '';
    if (!window.crypto || !crypto.subtle) return data.twoFaCode || '';
    const keyData = base32Decode(data.twoFaSecret);
    const key = await crypto.subtle.importKey('raw', keyData, {name:'HMAC', hash:'SHA-1'}, false, ['sign']);
    const counter = Math.floor(Date.now()/30000), msg = new ArrayBuffer(8), view = new DataView(msg);
    view.setUint32(4, counter, false);
    const sig = new Uint8Array(await crypto.subtle.sign('HMAC', key, msg));
    const offset = sig[sig.length-1] & 15;
    const value = (((sig[offset]&127)<<24)|(sig[offset+1]<<16)|(sig[offset+2]<<8)|sig[offset+3])>>>0;
    return String(value % 1000000).padStart(6, '0');
  }
  async function fillCredential(kind){
    const value = kind === 'username' ? data.username : (kind === 'password' ? data.password : await currentTOTP());
    if (!value) { toast('未配置内容'); return; }
    const target = targetFor(kind);
    if (setValue(target, value)) toast('已输入' + (kind === 'username' ? '账号' : kind === 'password' ? '密码' : '2FA'));
    else toast('请先点一下要输入的位置');
  }
  function addButton(text, kind){
    if ((kind === 'username' && !data.username) || (kind === 'password' && !data.password) || (kind === 'totp' && !data.twoFaSecret && !data.twoFaCode)) return null;
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.textContent = text;
    btn.style.cssText = 'height:28px;padding:0 10px;border:1px solid #d1d5db;border-radius:5px;background:#fff;color:#111827;font:12px -apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;cursor:pointer';
    btn.addEventListener('click', function(ev){ ev.preventDefault(); ev.stopPropagation(); fillCredential(kind); });
    return btn;
  }
  function mount(){
    if (document.getElementById(rootId)) return;
    const wrap = document.createElement('div');
    wrap.id = rootId;
    wrap.style.cssText = 'position:fixed;right:14px;bottom:18px;z-index:2147483647;display:flex;align-items:center;gap:6px;padding:8px;border:1px solid rgba(148,163,184,.5);border-radius:8px;background:rgba(248,250,252,.96);box-shadow:0 10px 28px rgba(15,23,42,.18);backdrop-filter:saturate(140%) blur(8px)';
    const title = document.createElement('span');
    title.textContent = data.platformName || '平台';
    title.style.cssText = 'max-width:120px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:#475569;font:12px -apple-system,BlinkMacSystemFont,Segoe UI,sans-serif';
    wrap.appendChild(title);
    [addButton('账号','username'), addButton('密码','password'), addButton('2FA','totp')].filter(Boolean).forEach(function(btn){ wrap.appendChild(btn); });
    document.documentElement.appendChild(wrap);
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount, { once: true });
  else mount();
})();`, nil
}

func (a *App) injectPlatformQuickInputAsync(profile *BrowserProfile, debugPort int) {
	if !shouldInjectPlatformQuickInput(profile) || debugPort <= 0 {
		return
	}
	snapshot := copyBrowserProfileSnapshot(profile)
	go a.injectPlatformQuickInputWithRetry(snapshot, debugPort)
}

func (a *App) injectPlatformQuickInputWithRetry(profile *BrowserProfile, debugPort int) {
	log := logger.New("Browser")
	deadline := time.Now().Add(30 * time.Second)
	injected := map[string]string{}
	for {
		if !a.isProfileRunningOnDebugPort(profile.ProfileId, debugPort) {
			return
		}

		count, err := injectPlatformQuickInput(debugPort, profile, injected)
		if err == nil && count > 0 {
			if time.Now().After(deadline) {
				return
			}
		} else if err != nil && time.Now().After(deadline) {
			log.Warn("平台快捷输入注入失败",
				logger.F("profile_id", profile.ProfileId),
				logger.F("platform", profile.Platform),
				logger.F("debug_port", debugPort),
				logger.F("error", err.Error()),
			)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func injectPlatformQuickInput(debugPort int, profile *BrowserProfile, injected map[string]string) (int, error) {
	script, err := renderPlatformQuickInputScript(profile)
	if err != nil {
		return 0, err
	}

	targets, err := fetchBrowserDebugTargets(debugPort)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, target := range targets {
		if !strings.EqualFold(strings.TrimSpace(target.Type), "page") || strings.TrimSpace(target.WebSocketDebuggerUrl) == "" {
			continue
		}
		if !platformQuickInputMatchesTargetURL(profile, target.URL) {
			continue
		}
		targetKey := strings.TrimSpace(target.ID)
		if targetKey == "" {
			targetKey = strings.TrimSpace(target.WebSocketDebuggerUrl)
		}
		injectedURL := injected[targetKey]
		if injectedURL == target.URL {
			continue
		}
		if err := cdpCallWebSocket(target.WebSocketDebuggerUrl, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": script}, 2*time.Second); err != nil {
			return count, err
		}
		if err := cdpCallWebSocket(target.WebSocketDebuggerUrl, "Runtime.evaluate", map[string]any{
			"expression":            script,
			"awaitPromise":          false,
			"includeCommandLineAPI": false,
		}, 2*time.Second); err != nil {
			return count, err
		}
		injected[targetKey] = target.URL
		count++
	}
	return count, nil
}

func platformQuickInputMatchesTargetURL(profile *BrowserProfile, targetURL string) bool {
	platformURL := browserProfilePlatformURL(profile)
	if platformURL == "" {
		return false
	}
	platformParsed, err := url.Parse(platformURL)
	if err != nil || platformParsed.Hostname() == "" {
		return false
	}
	targetParsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil || targetParsed.Hostname() == "" {
		return false
	}
	if targetParsed.Scheme != "http" && targetParsed.Scheme != "https" {
		return false
	}
	platformHost := strings.TrimPrefix(strings.ToLower(platformParsed.Hostname()), "www.")
	targetHost := strings.TrimPrefix(strings.ToLower(targetParsed.Hostname()), "www.")
	return targetHost == platformHost || strings.HasSuffix(targetHost, "."+platformHost)
}
