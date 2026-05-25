// 指纹参数序列化/反序列化工具

/**
 * 获取系统当前时区
 * @returns IANA 时区标识符，如 "Asia/Shanghai"
 */
export function getSystemTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
}

export interface FingerprintConfig {
  // 指纹种子（核心）
  seed?: string            // --fingerprint=<seed>  控制所有随机噪声的根种子

  // 基础身份
  brand?: string           // --fingerprint-brand=
  platform?: string        // --fingerprint-platform=
  deviceType?: string      // UA 生成辅助字段，不单独序列化
  osVersion?: string       // --fingerprint-platform-version=，同时用于 UA 生成
  browserMajor?: string    // UA 生成辅助字段，不单独序列化
  userAgent?: string       // --user-agent=
  lang?: string            // --lang=
  timezone?: string        // --timezone=

  // 屏幕与窗口
  resolution?: string      // --window-size=（预设值或 'custom'）
  customResolution?: string // 当 resolution === 'custom' 时使用
  colorDepth?: string      // --fingerprint-color-depth=

  // 硬件信息
  hardwareConcurrency?: string  // --fingerprint-hardware-concurrency=
  deviceMemory?: string         // --fingerprint-device-memory=

  // 渲染指纹
  canvasNoise?: boolean         // --fingerprint-canvas-noise=
  webglImageMode?: string       // --fingerprint-webgl-image=
  webglMetadataMode?: string    // --fingerprint-webgl-metadata=
  webglVendor?: string          // --fingerprint-gpu-vendor=
  webglRenderer?: string        // --fingerprint-gpu-renderer=
  webgpuMode?: string           // --fingerprint-webgpu=
  audioNoise?: boolean          // --fingerprint-audio-noise=
  clientRectsMode?: string      // --fingerprint-client-rects=
  speechVoicesMode?: string     // --fingerprint-speech-voices=
  deviceNameMode?: string       // --fingerprint-device-name=

  // 字体
  fonts?: string                // --fingerprint-fonts=

  // 网络与隐私
  webrtcPolicy?: string         // --webrtc-ip-handling-policy=
  geolocationPermission?: string // --geolocation-permission=
  geolocationBasedOnIp?: boolean // --geolocation-ip-based=
  portScanProtection?: boolean  // --fingerprint-port-scan-protection=
  cloudflareOptimize?: boolean  // --fingerprint-cloudflare-optimize=
  doNotTrack?: boolean          // --fingerprint-do-not-track=

  // 媒体设备
  mediaDevices?: string         // --fingerprint-media-devices= (格式: "2,1,0" 摄像头,麦克风,扬声器)

  // 触摸
  touchPoints?: string          // --fingerprint-touch-points=

  unknownArgs?: string[]        // 无法识别的原始参数，原样保留
}

export const PRESET_RESOLUTIONS = ['1920,1080', '1440,900', '1366,768', '2560,1440', '1280,800', '1600,900']
export const RANDOM_OPTION_VALUE = '__random__'

export const RANDOM_COLOR_DEPTHS = ['24', '24', '24', '30', '32']
export const RANDOM_HARDWARE_CONCURRENCY = ['4', '6', '8', '8', '10', '12', '16']
export const RANDOM_DEVICE_MEMORY = ['4', '8', '8', '16', '16', '32']
export const RANDOM_TOUCH_POINTS = ['0', '0', '0', '1', '5']

export const DEFAULT_FINGERPRINT_CONFIG: Partial<FingerprintConfig> = {
  brand: 'Chrome',
  platform: 'windows',
  deviceType: 'desktop',
  osVersion: '11.0',
  browserMajor: '139',
  lang: 'ip',
  timezone: 'ip',
  resolution: '1920,1080',
  colorDepth: '24',
  hardwareConcurrency: '8',
  deviceMemory: '8',
  touchPoints: '0',
  webglImageMode: 'random',
  webglMetadataMode: 'random',
  webglVendor: 'random',
  webglRenderer: 'random',
  webgpuMode: 'webgl',
  canvasNoise: true,
  audioNoise: true,
  clientRectsMode: 'random',
  speechVoicesMode: 'random',
  deviceNameMode: 'random',
  fonts: 'Arial,Helvetica,Times New Roman,Courier New,Verdana',
  webrtcPolicy: 'disable_non_proxied_udp',
  geolocationPermission: 'ask',
  geolocationBasedOnIp: true,
  portScanProtection: false,
  cloudflareOptimize: false,
  doNotTrack: false,
  mediaDevices: '2,1,1',
}

export function withFingerprintDefaults(config: FingerprintConfig): FingerprintConfig {
  const next: FingerprintConfig = {
    ...DEFAULT_FINGERPRINT_CONFIG,
    ...config,
    unknownArgs: config.unknownArgs ?? [],
  }
  if (!next.userAgent) {
    next.userAgent = buildUserAgent(next)
  }
  return next
}

// CLI 参数前缀 → FingerprintConfig 字段映射
export const KEY_MAP: Record<string, keyof FingerprintConfig> = {
  '--fingerprint': 'seed',
  '--fingerprint-brand': 'brand',
  '--fingerprint-platform': 'platform',
  '--fingerprint-platform-version': 'osVersion',
  '--user-agent': 'userAgent',
  '--lang': 'lang',
  '--timezone': 'timezone',
  '--window-size': 'resolution',
  '--fingerprint-color-depth': 'colorDepth',
  '--fingerprint-hardware-concurrency': 'hardwareConcurrency',
  '--fingerprint-device-memory': 'deviceMemory',
  '--fingerprint-canvas-noise': 'canvasNoise',
  '--fingerprint-webgl-image': 'webglImageMode',
  '--fingerprint-webgl-metadata': 'webglMetadataMode',
  '--fingerprint-webgl-vendor': 'webglVendor',
  '--fingerprint-webgl-renderer': 'webglRenderer',
  '--fingerprint-gpu-vendor': 'webglVendor',
  '--fingerprint-gpu-renderer': 'webglRenderer',
  '--fingerprint-webgpu': 'webgpuMode',
  '--fingerprint-audio-noise': 'audioNoise',
  '--fingerprint-client-rects': 'clientRectsMode',
  '--fingerprint-speech-voices': 'speechVoicesMode',
  '--fingerprint-device-name': 'deviceNameMode',
  '--fingerprint-fonts': 'fonts',
  '--webrtc-ip-handling-policy': 'webrtcPolicy',
  '--geolocation-permission': 'geolocationPermission',
  '--geolocation-ip-based': 'geolocationBasedOnIp',
  '--fingerprint-port-scan-protection': 'portScanProtection',
  '--fingerprint-cloudflare-optimize': 'cloudflareOptimize',
  '--fingerprint-do-not-track': 'doNotTrack',
  '--fingerprint-media-devices': 'mediaDevices',
  '--fingerprint-touch-points': 'touchPoints',
}

// FingerprintConfig → string[]
export function serialize(config: FingerprintConfig): string[] {
  config = withFingerprintDefaults(config)
  const args: string[] = []
  if (config.seed) args.push(`--fingerprint=${config.seed}`)
  if (config.brand) args.push(`--fingerprint-brand=${config.brand}`)
  if (config.platform) args.push(`--fingerprint-platform=${config.platform}`)
  if (config.osVersion) args.push(`--fingerprint-platform-version=${config.osVersion}`)
  if (config.userAgent) args.push(`--user-agent=${config.userAgent}`)
  if (config.lang) args.push(`--lang=${config.lang}`)
  if (config.timezone) {
    // 如果是 system，替换为实际系统时区
    const tz = config.timezone === 'system' ? getSystemTimezone() : config.timezone
    args.push(`--timezone=${tz}`)
  }

  const res = config.resolution === 'custom' ? config.customResolution : config.resolution
  if (res) args.push(`--window-size=${res}`)

  if (config.colorDepth) args.push(`--fingerprint-color-depth=${config.colorDepth}`)
  if (config.hardwareConcurrency) args.push(`--fingerprint-hardware-concurrency=${config.hardwareConcurrency}`)
  if (config.deviceMemory) args.push(`--fingerprint-device-memory=${config.deviceMemory}`)

  if (config.canvasNoise !== undefined) args.push(`--fingerprint-canvas-noise=${config.canvasNoise}`)
  if (config.webglImageMode) args.push(`--fingerprint-webgl-image=${config.webglImageMode}`)
  if (config.webglMetadataMode) args.push(`--fingerprint-webgl-metadata=${config.webglMetadataMode}`)
  if (config.webglVendor && config.webglVendor !== 'random') args.push(`--fingerprint-gpu-vendor=${config.webglVendor}`)
  if (config.webglRenderer && config.webglRenderer !== 'random') args.push(`--fingerprint-gpu-renderer=${config.webglRenderer}`)
  if (config.webgpuMode) args.push(`--fingerprint-webgpu=${config.webgpuMode}`)
  if (config.audioNoise !== undefined) args.push(`--fingerprint-audio-noise=${config.audioNoise}`)
  if (config.clientRectsMode) args.push(`--fingerprint-client-rects=${config.clientRectsMode}`)
  if (config.speechVoicesMode) args.push(`--fingerprint-speech-voices=${config.speechVoicesMode}`)
  if (config.deviceNameMode) args.push(`--fingerprint-device-name=${config.deviceNameMode}`)

  if (config.fonts) args.push(`--fingerprint-fonts=${config.fonts}`)

  if (config.webrtcPolicy) args.push(`--webrtc-ip-handling-policy=${config.webrtcPolicy}`)
  if (config.geolocationPermission) args.push(`--geolocation-permission=${config.geolocationPermission}`)
  if (config.geolocationBasedOnIp !== undefined) args.push(`--geolocation-ip-based=${config.geolocationBasedOnIp}`)
  if (config.portScanProtection !== undefined) args.push(`--fingerprint-port-scan-protection=${config.portScanProtection}`)
  if (config.cloudflareOptimize !== undefined) args.push(`--fingerprint-cloudflare-optimize=${config.cloudflareOptimize}`)
  if (config.doNotTrack !== undefined) args.push(`--fingerprint-do-not-track=${config.doNotTrack}`)
  if (config.mediaDevices) args.push(`--fingerprint-media-devices=${config.mediaDevices}`)
  if (config.touchPoints) args.push(`--fingerprint-touch-points=${config.touchPoints}`)

  return [...args, ...(config.unknownArgs ?? [])]
}

// string[] → FingerprintConfig
export function deserialize(args: string[]): FingerprintConfig {
  const config: FingerprintConfig = { unknownArgs: [] }

  for (const arg of args) {
    const eqIdx = arg.indexOf('=')
    if (eqIdx === -1) {
      config.unknownArgs!.push(arg)
      continue
    }
    const key = arg.slice(0, eqIdx)
    const val = arg.slice(eqIdx + 1)
    const field = KEY_MAP[key]

    if (!field) {
      config.unknownArgs!.push(arg)
      continue
    }

    if (field === 'canvasNoise' || field === 'audioNoise' || field === 'doNotTrack' || field === 'geolocationBasedOnIp' || field === 'portScanProtection' || field === 'cloudflareOptimize') {
      (config as Record<string, unknown>)[field] = val === 'true'
    } else if (field === 'resolution') {
      if (PRESET_RESOLUTIONS.includes(val)) {
        config.resolution = val
      } else {
        config.resolution = 'custom'
        config.customResolution = val
      }
    } else if (field === 'userAgent') {
      config.userAgent = val
      const uaInfo = parseUserAgent(val)
      config.deviceType = uaInfo.deviceType
      config.platform = config.platform || normalizePlatformValue(uaInfo.platform)
      config.osVersion = config.osVersion || uaInfo.osVersion
      config.browserMajor = uaInfo.browserMajor
      config.brand = config.brand || uaInfo.brand
    } else {
      (config as Record<string, unknown>)[field] = val
    }
  }

  return withFingerprintDefaults(config)
}

export function buildUserAgent(config: Partial<FingerprintConfig>): string {
  const brand = config.brand || 'Chrome'
  const platform = normalizePlatformValue(config.platform || 'windows')
  const deviceType = config.deviceType || 'desktop'
  const major = config.browserMajor || '139'
  const osVersion = config.osVersion || defaultOSVersion(platform, deviceType)

  if (deviceType === 'mobile') {
    if (platform === 'macos') {
      return `Mozilla/5.0 (iPhone; CPU iPhone OS ${osVersion || '17_5'} like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1`
    }
    const chromePart = `Chrome/${major}.0.0.0 Mobile Safari/537.36`
    return `Mozilla/5.0 (Linux; Android ${osVersion || '14'}; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) ${chromePart}${brand === 'Edge' ? ` EdgA/${major}.0.0.0` : ''}`
  }

  if (brand === 'Safari') {
    const macVersion = platform === 'macos' ? osVersion || '14_5' : '14_5'
    return `Mozilla/5.0 (Macintosh; Intel Mac OS X ${macVersion}) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15`
  }

  const osPart = platformUserAgentPart(platform, osVersion)
  if (brand === 'Firefox') {
    return `Mozilla/5.0 (${osPart}; rv:${major}.0) Gecko/20100101 Firefox/${major}.0`
  }
  const chromePart = `Chrome/${major}.0.0.0 Safari/537.36`
  return `Mozilla/5.0 (${osPart}) AppleWebKit/537.36 (KHTML, like Gecko) ${chromePart}${brand === 'Edge' ? ` Edg/${major}.0.0.0` : ''}`
}

export function defaultOSVersion(platform?: string, deviceType = 'desktop'): string {
  platform = normalizePlatformValue(platform)
  if (deviceType === 'mobile') {
    return platform === 'macos' ? '17_5' : '14'
  }
  if (platform === 'macos') return '14_5'
  if (platform === 'linux') return 'x86_64'
  return '11.0'
}

function platformUserAgentPart(platform: string, osVersion?: string): string {
  platform = normalizePlatformValue(platform)
  if (platform === 'macos') return `Macintosh; Intel Mac OS X ${osVersion || '14_5'}`
  if (platform === 'linux') return 'X11; Linux x86_64'
  const windowsVersion = osVersion === '11.0' ? '10.0' : osVersion || '10.0'
  return `Windows NT ${windowsVersion}; Win64; x64`
}

function parseUserAgent(userAgent: string): Required<Pick<FingerprintConfig, 'brand' | 'platform' | 'deviceType' | 'osVersion' | 'browserMajor'>> {
  const chromeMajor = userAgent.match(/(?:Chrome|CriOS|Edg|EdgA|Firefox)\/(\d+)/)?.[1] || ''
  const safariMajor = userAgent.match(/Version\/(\d+)/)?.[1] || ''
  const platform = /iPhone|Macintosh/.test(userAgent)
    ? 'macos'
    : /Android/.test(userAgent)
      ? 'linux'
      : /Linux/.test(userAgent)
        ? 'linux'
        : 'windows'
  const brand = /Firefox/.test(userAgent) ? 'Firefox' : /Edg/.test(userAgent) ? 'Edge' : /Safari/.test(userAgent) && !/Chrome|CriOS/.test(userAgent) ? 'Safari' : 'Chrome'
  const deviceType = /Mobile|Android|iPhone/.test(userAgent) ? 'mobile' : 'desktop'
  const windowsVersion = userAgent.match(/Windows NT ([^;)]+)/)?.[1]
  const macVersion = userAgent.match(/(?:Mac OS X|CPU iPhone OS) ([^;) ]+)/)?.[1]
  const androidVersion = userAgent.match(/Android ([^;)]+)/)?.[1]
  return {
    brand,
    platform,
    deviceType,
    osVersion: windowsVersion || macVersion || androidVersion || defaultOSVersion(platform, deviceType),
    browserMajor: chromeMajor || safariMajor || '139',
  }
}

function normalizePlatformValue(platform?: string): string {
  return platform === 'mac' ? 'macos' : platform || 'windows'
}

// 生成随机指纹种子（32位正整数）
export function randomFingerprintSeed(): string {
  return String(Math.floor(Math.random() * 2147483647) + 1)
}

function randomChoice<T>(items: T[]): T {
  return items[Math.floor(Math.random() * items.length)]
}

export function randomScreenHardwarePatch(): Pick<FingerprintConfig, 'resolution' | 'customResolution' | 'colorDepth' | 'hardwareConcurrency' | 'deviceMemory' | 'touchPoints'> {
  return {
    resolution: randomChoice(PRESET_RESOLUTIONS),
    customResolution: undefined,
    colorDepth: randomChoice(RANDOM_COLOR_DEPTHS),
    hardwareConcurrency: randomChoice(RANDOM_HARDWARE_CONCURRENCY),
    deviceMemory: randomChoice(RANDOM_DEVICE_MEMORY),
    touchPoints: randomChoice(RANDOM_TOUCH_POINTS),
  }
}

export function createRandomizedFingerprintConfig(base: FingerprintConfig | Partial<FingerprintConfig> = {}): FingerprintConfig {
  const next = withFingerprintDefaults({
    ...base,
    ...randomScreenHardwarePatch(),
    seed: base.seed || randomFingerprintSeed(),
  })
  next.userAgent = base.userAgent || buildUserAgent(next)
  return next
}

// ─── 预设指纹配置 ────────────────────────────────────────────────────────────

export interface FingerprintPreset {
  id: string
  name: string
  description: string
  config: Partial<FingerprintConfig>
}

export const FINGERPRINT_PRESETS: FingerprintPreset[] = [
  {
    id: 'win-chrome-office',
    name: 'Windows / Chrome / 办公',
    description: '模拟国内办公室 Windows 用户，中文环境，1920x1080',
    config: {
      brand: 'Chrome',
      platform: 'windows',
      lang: 'zh-CN',
      timezone: 'Asia/Shanghai',
      resolution: '1920,1080',
      colorDepth: '24',
      hardwareConcurrency: '8',
      deviceMemory: '8',
      canvasNoise: true,
      audioNoise: true,
      webglVendor: 'Intel',
      webglRenderer: 'Intel(R) UHD Graphics 630',
      fonts: 'Arial,Microsoft YaHei,SimSun,SimHei,Helvetica,Times New Roman',
      webrtcPolicy: 'disable_non_proxied_udp',
      doNotTrack: false,
      touchPoints: '0',
    },
  },
  {
    id: 'win-chrome-gaming',
    name: 'Windows / Chrome / 游戏主机',
    description: '模拟高配游戏 PC，NVIDIA 显卡，2560x1440',
    config: {
      brand: 'Chrome',
      platform: 'windows',
      lang: 'en-US',
      timezone: 'America/New_York',
      resolution: '2560,1440',
      colorDepth: '24',
      hardwareConcurrency: '16',
      deviceMemory: '16',
      canvasNoise: true,
      audioNoise: true,
      webglVendor: 'NVIDIA',
      webglRenderer: 'NVIDIA GeForce RTX 3080',
      fonts: 'Arial,Helvetica,Times New Roman,Courier New,Verdana',
      webrtcPolicy: 'disable_non_proxied_udp',
      doNotTrack: false,
      touchPoints: '0',
    },
  },
  {
    id: 'mac-chrome-designer',
    name: 'macOS / Chrome / 设计师',
    description: '模拟 Mac 设计师用户，Apple GPU，Retina 分辨率',
    config: {
      brand: 'Chrome',
      platform: 'macos',
      lang: 'zh-CN',
      timezone: 'Asia/Shanghai',
      resolution: '2560,1440',
      colorDepth: '30',
      hardwareConcurrency: '10',
      deviceMemory: '16',
      canvasNoise: true,
      audioNoise: true,
      webglVendor: 'Apple',
      webglRenderer: 'Apple M2',
      fonts: 'Arial,Helvetica,PingFang SC,Hiragino Sans GB,STHeiti,Times New Roman',
      webrtcPolicy: 'disable_non_proxied_udp',
      doNotTrack: true,
      touchPoints: '0',
    },
  },
  {
    id: 'win-edge-enterprise',
    name: 'Windows / Edge / 企业',
    description: '模拟企业 Windows 用户，Edge 浏览器，标准配置',
    config: {
      brand: 'Edge',
      platform: 'windows',
      lang: 'zh-CN',
      timezone: 'Asia/Shanghai',
      resolution: '1366,768',
      colorDepth: '24',
      hardwareConcurrency: '4',
      deviceMemory: '4',
      canvasNoise: true,
      audioNoise: false,
      webglVendor: 'Intel',
      webglRenderer: 'Intel(R) HD Graphics 520',
      fonts: 'Arial,Microsoft YaHei,Calibri,Segoe UI,Times New Roman',
      webrtcPolicy: 'default_public_interface_only',
      doNotTrack: false,
      touchPoints: '0',
    },
  },
  {
    id: 'win-chrome-us-user',
    name: 'Windows / Chrome / 美国用户',
    description: '模拟美国普通用户，英文环境，AMD 显卡',
    config: {
      brand: 'Chrome',
      platform: 'windows',
      lang: 'en-US',
      timezone: 'America/Los_Angeles',
      resolution: '1920,1080',
      colorDepth: '24',
      hardwareConcurrency: '8',
      deviceMemory: '8',
      canvasNoise: true,
      audioNoise: true,
      webglVendor: 'AMD',
      webglRenderer: 'AMD Radeon RX 6600',
      fonts: 'Arial,Helvetica,Times New Roman,Courier New,Georgia',
      webrtcPolicy: 'disable_non_proxied_udp',
      doNotTrack: false,
      touchPoints: '0',
    },
  },
  {
    id: 'mac-safari-jp',
    name: 'macOS / Safari / 日本用户',
    description: '模拟日本 Mac 用户，Safari 风格，日语环境',
    config: {
      brand: 'Safari',
      platform: 'macos',
      lang: 'ja-JP',
      timezone: 'Asia/Tokyo',
      resolution: '1440,900',
      colorDepth: '24',
      hardwareConcurrency: '8',
      deviceMemory: '8',
      canvasNoise: true,
      audioNoise: true,
      webglVendor: 'Apple',
      webglRenderer: 'Apple M1',
      fonts: 'Arial,Helvetica,Hiragino Kaku Gothic ProN,Yu Gothic,Times New Roman',
      webrtcPolicy: 'disable_non_proxied_udp',
      doNotTrack: true,
      touchPoints: '0',
    },
  },
  {
    id: 'win-chrome-uk-office',
    name: 'Windows / Chrome / 英国-办公',
    description: '模拟英国办公室 Windows 用户，英文环境 (en-GB)',
    config: {
      brand: 'Chrome',
      platform: 'windows',
      lang: 'en-GB',
      timezone: 'Europe/London',
      resolution: '1920,1080',
      colorDepth: '24',
      hardwareConcurrency: '8',
      deviceMemory: '8',
      canvasNoise: true,
      audioNoise: true,
      webglVendor: 'Intel',
      webglRenderer: 'Intel(R) UHD Graphics 630',
      fonts: 'Arial,Helvetica,Times New Roman,Courier New,Verdana',
      webrtcPolicy: 'disable_non_proxied_udp',
      doNotTrack: false,
      touchPoints: '0',
    },
  },
  {
    id: 'mac-chrome-us-edu',
    name: 'macOS / Chrome / 美国-教育',
    description: '模拟美国大学教育网 Mac 用户，英文环境 (en-US)',
    config: {
      brand: 'Chrome',
      platform: 'macos',
      lang: 'en-US',
      timezone: 'America/New_York',
      resolution: '1440,900',
      colorDepth: '24',
      hardwareConcurrency: '8',
      deviceMemory: '8',
      canvasNoise: true,
      audioNoise: true,
      webglVendor: 'Apple',
      webglRenderer: 'Apple M1',
      fonts: 'Arial,Helvetica,Times New Roman,Courier New,Georgia',
      webrtcPolicy: 'disable_non_proxied_udp',
      doNotTrack: false,
      touchPoints: '0',
    },
  },
]
