package backend

import (
	"ant-chrome/backend/internal/config"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type browserStartPageModel struct {
	Title       string
	Serial      string
	ProfileName string
	Username    string
	Password    string
	Platform    string
	PlatformURL string
	GroupName   string
	Tags        string
	Keywords    string
	UserAgent   string
	TwoFASecret string
	TwoFACode   string
	TwoFAError  string
	StartedAt   string
	Language    string
	Timezone    string
}

func shouldUseBrowserStartPage(startURLs []string, defaultStartURLs []string, skipDefaultStartURLs bool, restoreLastSession bool) bool {
	return len(normalizeNonEmptyStrings(startURLs)) == 0 &&
		len(normalizeNonEmptyStrings(defaultStartURLs)) == 0 &&
		!skipDefaultStartURLs &&
		!restoreLastSession
}

func (a *App) browserDefaultLaunchTargets(input browserStartInput, profile *BrowserProfile, restoreLastSession bool, launchedAt time.Time) ([]string, error) {
	defaultStartURLs := a.browserDefaultStartURLs()
	if !shouldUseProfilePlatformPage(input.StartURLs, input.SkipDefaultStartURLs, restoreLastSession) {
		return defaultStartURLs, nil
	}

	startPageURL, err := a.browserStartPageURL(profile, launchedAt)
	if err != nil {
		return defaultStartURLs, err
	}

	targets := []string{startPageURL}
	if platformURL := browserProfilePlatformURL(profile); platformURL != "" {
		targets = append(targets, platformURL)
	}
	return targets, nil
}

func (a *App) browserStartPageURL(profile *BrowserProfile, launchedAt time.Time) (string, error) {
	if profile == nil {
		return "", os.ErrInvalid
	}

	dir := a.resolveAppPath(filepath.ToSlash(filepath.Join("data", "runtime", "start-pages")))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	fileName := safeStartPageFileName(browserStartPageSerial(profile)) + ".html"
	pagePath := filepath.Join(dir, fileName)
	html, err := renderBrowserStartPageHTML(a.browserStartPageModel(profile, launchedAt))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(pagePath, []byte(html), 0o644); err != nil {
		return "", err
	}
	if a.launchServer != nil {
		a.launchServer.SetStartPageDir(dir)
		if startPageURL := a.launchServer.StartPageURL(fileName); startPageURL != "" {
			return startPageURL, nil
		}
	}
	return a.browserStartPageServiceURL(fileName), nil
}

func (a *App) browserStartPageServiceURL(fileName string) string {
	port := config.DefaultLaunchServerPort
	if a != nil {
		if a.launchServer != nil && a.launchServer.Port() > 0 {
			port = a.launchServer.Port()
		} else if a.config != nil && a.config.LaunchServer.Port > 0 {
			port = a.config.LaunchServer.Port
		}
	}
	return "http://127.0.0.1:" + strconv.Itoa(port) + "/start-pages/" + url.PathEscape(filepath.Base(fileName))
}

func (a *App) browserStartPageModel(profile *BrowserProfile, launchedAt time.Time) browserStartPageModel {
	profileName := strings.TrimSpace(profile.ProfileName)
	if profileName == "" {
		profileName = strings.TrimSpace(profile.ProfileId)
	}
	startedAt := launchedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	serial := browserStartPageSerial(profile)
	twoFASecret := normalizeTOTPSecret(profile.TwoFASecret)
	twoFACode := ""
	twoFAError := ""
	if twoFASecret != "" {
		var err error
		twoFACode, err = browserStartPageTOTPCode(twoFASecret, time.Now())
		if err != nil {
			twoFAError = "2FA密钥无效"
		}
	}

	return browserStartPageModel{
		Title:       strings.TrimSpace(serial + " " + profileName),
		Serial:      serial,
		ProfileName: profileName,
		Username:    browserStartPageUsername(profile, profileName),
		Password:    strings.TrimSpace(profile.Password),
		Platform:    browserStartPagePlatformName(profile),
		PlatformURL: browserProfilePlatformURL(profile),
		GroupName:   a.browserStartPageGroupName(profile),
		Tags:        strings.Join(normalizeNonEmptyStrings(profile.Tags), ", "),
		Keywords:    strings.Join(normalizeNonEmptyStrings(profile.Keywords), ", "),
		UserAgent:   browserStartPageUserAgent(profile),
		TwoFASecret: twoFASecret,
		TwoFACode:   twoFACode,
		TwoFAError:  twoFAError,
		StartedAt:   startedAt.Format("2006-01-02 15:04:05"),
		Language:    browserStartPageLanguage(profile),
		Timezone:    browserStartPageArgValue(profile.FingerprintArgs, "--timezone"),
	}
}

func shouldUseProfilePlatformPage(startURLs []string, skipDefaultStartURLs bool, restoreLastSession bool) bool {
	return len(normalizeNonEmptyStrings(startURLs)) == 0 && !skipDefaultStartURLs && !restoreLastSession
}

func browserProfilePlatformURL(profile *BrowserProfile) string {
	if profile == nil {
		return ""
	}
	if strings.TrimSpace(profile.Platform) == "" || strings.EqualFold(strings.TrimSpace(profile.Platform), "none") {
		return ""
	}
	value := strings.TrimSpace(profile.PlatformURL)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return parsed.String()
	}
	return "https://" + strings.TrimLeft(value, "/")
}

func browserStartPagePlatformName(profile *BrowserProfile) string {
	if profile == nil {
		return ""
	}
	if strings.TrimSpace(profile.Platform) == "" || strings.EqualFold(strings.TrimSpace(profile.Platform), "none") {
		return ""
	}
	if value := strings.TrimSpace(profile.PlatformName); value != "" {
		return value
	}
	return strings.TrimSpace(profile.Platform)
}

func browserStartPageUsername(profile *BrowserProfile, profileName string) string {
	if profile == nil {
		return strings.TrimSpace(profileName)
	}
	if value := strings.TrimSpace(profile.Username); value != "" {
		return value
	}
	return strings.TrimSpace(profileName)
}

func browserStartPageSerial(profile *BrowserProfile) string {
	if profile == nil {
		return ""
	}
	if profile.ID > 0 {
		return strconv.FormatInt(profile.ID, 10)
	}
	return strings.TrimSpace(profile.ProfileId)
}

func (a *App) browserStartPageGroupName(profile *BrowserProfile) string {
	groupID := strings.TrimSpace(profile.GroupId)
	if groupID == "" {
		return ""
	}
	if a != nil && a.browserMgr != nil && a.browserMgr.GroupDAO != nil {
		if group, err := a.browserMgr.GroupDAO.GetById(groupID); err == nil && group != nil {
			if name := strings.TrimSpace(group.GroupName); name != "" {
				return name
			}
		}
	}
	return groupID
}

func browserStartPageLanguage(profile *BrowserProfile) string {
	if profile == nil {
		return ""
	}
	if value := browserStartPageArgValue(profile.LaunchArgs, "--accept-lang", "--accept-language"); value != "" {
		return value
	}
	return browserStartPageArgValue(profile.FingerprintArgs, "--lang")
}

func browserStartPageUserAgent(profile *BrowserProfile) string {
	if profile == nil {
		return ""
	}
	if value := browserStartPageArgValue(profile.LaunchArgs, "--user-agent"); value != "" {
		return value
	}
	return browserStartPageArgValue(profile.FingerprintArgs, "--user-agent")
}

func browserStartPageArgValue(args []string, keys ...string) string {
	for _, arg := range args {
		value := strings.TrimSpace(arg)
		for _, key := range keys {
			prefix := key + "="
			if strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) {
				return strings.TrimSpace(value[len(prefix):])
			}
		}
	}
	return ""
}

func normalizeTOTPSecret(secret string) string {
	value := strings.TrimSpace(secret)
	if strings.HasPrefix(strings.ToLower(value), "otpauth://") {
		if parsed, err := url.Parse(value); err == nil {
			value = parsed.Query().Get("secret")
		}
	}
	upper := strings.ToUpper(strings.TrimSpace(value))
	upper = strings.ReplaceAll(upper, " ", "")
	upper = strings.ReplaceAll(upper, "-", "")
	return upper
}

func browserStartPageTOTPCode(secret string, now time.Time) (string, error) {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalizeTOTPSecret(secret))
	if err != nil {
		return "", err
	}
	counter := uint64(now.Unix() / 30)
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)

	mac := hmac.New(sha1.New, decoded)
	if _, err := mac.Write(msg[:]); err != nil {
		return "", err
	}
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)
	return strconv.FormatInt(int64(value%1000000)+1000000, 10)[1:], nil
}

func safeStartPageFileName(profileID string) string {
	value := strings.TrimSpace(profileID)
	if value == "" {
		return "profile"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "profile"
	}
	return b.String()
}

func renderBrowserStartPageHTML(model browserStartPageModel) (string, error) {
	if strings.TrimSpace(model.Title) == "" {
		model.Title = "Ant Browser"
	}
	var buf bytes.Buffer
	if err := browserStartPageTemplate.Execute(&buf, model); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var browserStartPageTemplate = template.Must(template.New("browser-start-page").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
  <meta http-equiv="X-UA-Compatible" content="IE=edge">
  <meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
  <link rel="icon" href="data:image/ico;base64,aWNv">
  <title>{{.Title}}</title>
  <style>
    *{box-sizing:border-box;margin:0;padding:0;font-size:9pt}
    body{font-family:Helvetica Neue,Helvetica,Microsoft YaHei,PingFang SC,Hiragino Sans GB,Arial,sans-serif;background:#f0f4fa;color:#585a6f}
    .open-info .box{width:900px;max-width:95%;margin:16px auto;background:#fff;box-shadow:0 0 10px rgba(0,0,0,.28)}
    .open-ip{position:relative;text-align:center;color:#fff;background:#8eb7c6}
    .open-ip .ip{min-height:98px;padding:36px 10px 20px;font-size:52px;font-weight:300;line-height:1.05;word-break:break-word}
    .open-ip .ip span{font-size:52px;font-weight:300;line-height:1.05}
    .open-ip .ip .ip-fail{font-size:14px}
    .ping{padding:5px 10px 10px;text-align:center}
    .ping a{display:inline-block;margin:0 3px;padding:0 2px;border-radius:2px;color:#fff;text-decoration:none}
    .ping a::before{display:inline-block;width:6px;height:6px;margin-right:2px;border-radius:50%;background:#bec5ff;content:"";vertical-align:middle}
    .ping a.success::before{background:#05ff05}.ping a.fail::before{background:#ff3f3f}
    .locales{padding:16px 10px 10px;border-top:1px solid hsla(0,0%,100%,.2);background:rgba(0,0,0,.2);box-shadow:0 -15px 15px hsla(0,0%,100%,.1)}
    .locales span{display:inline-block;margin:0 10px;color:#ccc}.locales i{color:#fff;font-style:normal;font-weight:600;font-size:14px}
    .content{padding:20px}
    .row-left{overflow:hidden;padding:10px;line-height:32px}
    .row-left .bd{float:left;width:95px;padding-right:10px;text-align:right;color:#999}
    .row-left .hd{min-height:32px;margin-left:110px;padding:4px 10px;border-bottom:1px solid #eee;font-size:14px;line-height:24px;word-break:break-word}
    .row-left .hd .no-re{color:#ccc;font-style:italic}
    .twofa-code{display:inline-block;min-width:76px;font-size:18px;font-weight:700;letter-spacing:1px;color:#4f5870}
    .twofa-timer{display:inline-block;margin-left:8px;color:#a3a8b5}
    .copy{display:inline-flex;margin-left:8px;padding:0 8px;border:1px solid #d8deea;border-radius:3px;background:#f7f9fd;color:#64748b;cursor:pointer}
    .platform-link{display:inline-block;margin-left:10px;color:#3b82f6;text-decoration:none}
    .version{text-align:center;color:#aaa;padding:8px}
    #toast-container{position:fixed;top:20px;right:20px;z-index:9999}
    .toast{min-width:250px;margin-bottom:10px;padding:15px;border:1px solid #e1f3d8;border-radius:4px;background:#f0f9eb;color:#67c23a;box-shadow:0 2px 10px rgba(0,0,0,.2);opacity:0;transform:translateX(100%);transition:all .3s ease}
    .toast.show{opacity:1;transform:translateX(0)}
    @media screen and (max-width:600px){.open-ip .ip,.open-ip .ip span{font-size:35px}.ping a span.all{display:none}.ping a span.first{display:inline}.row-left .bd{float:none;width:auto;text-align:left}.row-left .hd{margin-left:0}}
    @media screen and (min-width:601px){.ping a span.first{display:none}}
  </style>
</head>
<body>
<div class="open-info">
  <div class="box">
    <div class="open-ip">
      <div class="ip"><span id="ip-ip">----</span></div>
      <div class="ping">
        <a href="https://www.google.com/" target="_blank" rel="noopener noreferrer" id="ping_0"><span class="all">Google</span><span class="first">GG</span></a>
        <a href="https://www.wikipedia.org/" target="_blank" rel="noopener noreferrer" id="ping_1"><span class="all">Wikipedia</span><span class="first">Wiki</span></a>
        <a href="https://www.facebook.com/" target="_blank" rel="noopener noreferrer" id="ping_2"><span class="all">Facebook</span><span class="first">FB</span></a>
        <a href="https://www.tiktok.com/" target="_blank" rel="noopener noreferrer" id="ping_3"><span class="all">Tiktok</span><span class="first">Tiktok</span></a>
        <a href="https://www.amazon.com/" target="_blank" rel="noopener noreferrer" id="ping_4"><span class="all">Amazon</span><span class="first">Amz</span></a>
        <a href="https://whoer.net/" target="_blank" rel="noopener noreferrer" id="ping_5"><span class="all">Whoer</span><span class="first">Wh</span></a>
      </div>
      <div class="locales">
        <div>IP-API（仅供参考）:&nbsp;<span><i id="country"></i></span>&nbsp;/&nbsp;<span><i id="region"></i></span>&nbsp;/&nbsp;<span><i id="city"></i></span>&nbsp;/&nbsp;<span><i id="timezone"></i></span></div>
        <div style="margin-top:5px;">经纬度：<span><i id="lon">-</i></span>/<span><i id="lat">-</i></span>&nbsp;&nbsp;邮编：<span><i id="zip">-</i></span></div>
      </div>
    </div>
    <div class="content">
      <div class="row-left"><div class="bd">序号:</div><div class="hd">{{.Serial}}</div></div>
      <div class="row-left"><div class="bd">实例名称:</div><div class="hd">{{.ProfileName}}</div></div>
      <div class="row-left"><div class="bd">用户名:</div><div class="hd">{{.Username}}<button class="copy" onclick="copyText({{printf "%q" .Username}})">复制</button></div></div>
      <div class="row-left"><div class="bd">密码:</div><div class="hd">{{if .Password}}{{.Password}}<button class="copy" onclick="copyText({{printf "%q" .Password}})">复制</button>{{else}}<span class="no-re">未设置密码</span>{{end}}</div></div>
      <div class="row-left"><div class="bd">平台:</div><div class="hd">{{if .Platform}}{{.Platform}}{{if .PlatformURL}}<a class="platform-link" href="{{.PlatformURL}}" target="_blank" rel="noopener noreferrer">{{.PlatformURL}}</a>{{end}}{{else}}<span class="no-re">无平台</span>{{end}}</div></div>
      <div class="row-left"><div class="bd">2FA验证码:</div><div class="hd">{{if .TwoFASecret}}<span id="twofa-code" class="twofa-code" data-secret="{{.TwoFASecret}}">{{if .TwoFAError}}{{.TwoFAError}}{{else}}{{.TwoFACode}}{{end}}</span><span id="twofa-timer" class="twofa-timer"></span><button class="copy" onclick="copyTOTP()">复制</button>{{else}}<span class="no-re">未设置2FA密钥</span>{{end}}</div></div>
      <div class="row-left"><div class="bd">分组:</div><div class="hd">{{.GroupName}}</div></div>
      <div class="row-left"><div class="bd">标签:</div><div class="hd">{{.Tags}}</div></div>
      <div class="row-left"><div class="bd">关键词:</div><div class="hd">{{.Keywords}}</div></div>
      <div class="row-left"><div class="bd">启动时间:</div><div class="hd">{{.StartedAt}}</div></div>
      <div class="finger">
        <div class="row-left"><div class="bd">语言:</div><div class="hd">{{.Language}}</div></div>
        <div class="row-left"><div class="bd">UserAgent:</div><div class="hd" id="user-agent">{{.UserAgent}}</div></div>
        <div class="row-left"><div class="bd">时区:</div><div class="hd">{{.Timezone}}</div></div>
      </div>
    </div>
  </div>
  <div class="version">Ant Browser</div>
</div>
<script>
function setText(id,value){var el=document.getElementById(id);if(el)el.textContent=value||''}
function requestJSON(url){
  return new Promise(function(resolve,reject){
    var xhr = new XMLHttpRequest(); xhr.timeout = 10000; xhr.open('GET', url); xhr.send();
    xhr.onload = function(){ if(xhr.status >= 200 && xhr.status < 300){ try{resolve(JSON.parse(xhr.responseText))}catch(e){reject(e)} } else { reject(new Error('HTTP '+xhr.status)) } };
    xhr.onerror = reject; xhr.ontimeout = reject;
  })
}
function fillIP(data){
  var ip = data.query || data.ip || '';
  if(!ip){setText('ip-ip','Check Error');return}
  setText('ip-ip', ip);
  setText('country', data.country || data.countryCode || '');
  setText('region', data.regionName || data.region || '');
  setText('city', data.city || '');
  setText('timezone', data.timezone && data.timezone.id ? data.timezone.id : (data.timezone || ''));
  setText('lat', data.lat || data.latitude || '');
  setText('lon', data.lon || data.longitude || '');
  setText('zip', data.zip || data.postal || '');
}
function CheckIP(){
  requestJSON('http://www.ixbrowser.com/api/ip-api')
    .catch(function(){return requestJSON('http://ip-api.com/json/?fields=status,message,query,country,regionName,city,timezone,lat,lon,zip')})
    .then(function(data){ if(!data || data.status === 'fail' || data.success === false){setText('ip-ip','Check Error');return} fillIP(data) })
    .catch(function(){setText('ip-ip','Check Error')})
}
function checkWebSite(){
  [
    ['ping_0','https://www.google.com/favicon.ico'],
    ['ping_1','https://en.wikipedia.org/favicon.ico'],
    ['ping_2','https://www.facebook.com/favicon.ico'],
    ['ping_3','https://www.tiktok.com/favicon.ico'],
    ['ping_4','https://www.amazon.com/favicon.ico'],
    ['ping_5','https://whoer.net/favicon.ico']
  ].forEach(function(item){
    var img = new Image(); var el = document.getElementById(item[0]);
    img.referrerPolicy = 'no-referrer';
    img.onload = function(){ if(el) el.className='success' };
    img.onerror = function(){ if(el) el.className='fail' };
    img.src = item[1] + '?v=' + Date.now();
  })
}
async function copyText(text){
  try{ await navigator.clipboard.writeText(text); showToast('已复制到剪贴板') }
  catch(e){ showToast('复制失败') }
}
function base32Decode(value){
  var raw=value||'';
  if(/^otpauth:\/\//i.test(raw)){try{raw=new URL(raw).searchParams.get('secret')||''}catch(e){}}
  var alphabet='ABCDEFGHIJKLMNOPQRSTUVWXYZ234567', clean=raw.toUpperCase().replace(/[\s=-]/g,''), bits='', out=[];
  for(var i=0;i<clean.length;i++){var idx=alphabet.indexOf(clean[i]); if(idx<0) throw new Error('invalid base32'); bits += idx.toString(2).padStart(5,'0')}
  for(var j=0;j+8<=bits.length;j+=8){out.push(parseInt(bits.slice(j,j+8),2))}
  return new Uint8Array(out)
}
async function calculateTOTP(secret){
  if(!window.crypto || !crypto.subtle) throw new Error('crypto unavailable');
  var keyData=base32Decode(secret);
  var key=await crypto.subtle.importKey('raw', keyData, {name:'HMAC', hash:'SHA-1'}, false, ['sign']);
  var counter=Math.floor(Date.now()/30000), msg=new ArrayBuffer(8), view=new DataView(msg);
  view.setUint32(4, counter, false);
  var sig=new Uint8Array(await crypto.subtle.sign('HMAC', key, msg));
  var offset=sig[sig.length-1]&15;
  var value=(((sig[offset]&127)<<24)|(sig[offset+1]<<16)|(sig[offset+2]<<8)|sig[offset+3])>>>0;
  return String(value%1000000).padStart(6,'0')
}
async function updateTOTP(){
  var codeEl=document.getElementById('twofa-code'); if(!codeEl) return;
  var timerEl=document.getElementById('twofa-timer'), remain=30-Math.floor(Date.now()/1000)%30;
  if(timerEl) timerEl.textContent=remain+'s';
  try{ codeEl.textContent=await calculateTOTP(codeEl.getAttribute('data-secret')||'') }
  catch(e){ codeEl.textContent='2FA密钥无效'; if(timerEl) timerEl.textContent='' }
}
function copyTOTP(){
  var el=document.getElementById('twofa-code');
  copyText(el ? el.textContent.replace(/\D/g,'') : '')
}
function showToast(message, duration){
  var toast=document.createElement('div'); toast.className='toast'; toast.textContent=message;
  var container=document.getElementById('toast-container') || createToastContainer(); container.appendChild(toast);
  setTimeout(function(){toast.classList.add('show')},10);
  setTimeout(function(){toast.classList.remove('show');setTimeout(function(){toast.remove()},300)},duration||3000);
}
function createToastContainer(){var c=document.createElement('div');c.id='toast-container';document.body.appendChild(c);return c}
function fillNavigatorUserAgent(){var el=document.getElementById('user-agent'); if(el && !el.textContent.trim()){el.textContent=navigator.userAgent||''}}
fillNavigatorUserAgent(); CheckIP(); checkWebSite(); updateTOTP(); setInterval(updateTOTP, 1000);
</script>
</body>
</html>`))
