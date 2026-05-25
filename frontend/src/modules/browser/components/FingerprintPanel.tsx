import { useEffect, useState } from 'react'
import { ChevronDown, ChevronUp, RefreshCw, Wand2 } from 'lucide-react'
import { ConfirmModal, FormItem, Input, Select, Textarea } from '../../../shared/components'
import {
  type FingerprintConfig,
  FINGERPRINT_PRESETS,
  PRESET_RESOLUTIONS,
  RANDOM_OPTION_VALUE,
  buildUserAgent,
  defaultOSVersion,
  deserialize,
  getSystemTimezone,
  randomScreenHardwarePatch,
  randomFingerprintSeed,
  serialize,
} from '../utils/fingerprintSerializer'

interface FingerprintPanelProps {
  value: string[]
  onChange: (args: string[]) => void
}

const BRAND_OPTIONS = [
  { value: 'Chrome', label: 'Chrome' },
  { value: 'Edge', label: 'Edge' },
  { value: 'Firefox', label: 'Firefox' },
  { value: 'Safari', label: 'Safari' },
]

const PLATFORM_OPTIONS = [
  { value: 'windows', label: 'Windows' },
  { value: 'macos', label: 'macOS' },
  { value: 'linux', label: 'Linux' },
]

const DEVICE_TYPE_OPTIONS = [
  { value: 'desktop', label: '桌面' },
  { value: 'mobile', label: '移动' },
]

const BROWSER_MAJOR_OPTIONS = [
  { value: '139', label: '139' },
  { value: '138', label: '138' },
  { value: '137', label: '137' },
  { value: '136', label: '136' },
  { value: '135', label: '135' },
]

const OS_VERSION_OPTIONS: Record<string, { value: string; label: string }[]> = {
  windows: [
    { value: '11.0', label: 'Windows 11' },
    { value: '10.0', label: 'Windows 10' },
    { value: '6.3', label: 'Windows 8.1' },
    { value: '6.1', label: 'Windows 7' },
  ],
  macos: [
    { value: '14_5', label: 'macOS 14.5' },
    { value: '13_6', label: 'macOS 13.6' },
    { value: '12_7', label: 'macOS 12.7' },
  ],
  linux: [
    { value: 'x86_64', label: 'Linux x86_64' },
  ],
}

const MOBILE_OS_VERSION_OPTIONS: Record<string, { value: string; label: string }[]> = {
  mac: [
    { value: '17_5', label: 'iOS 17.5' },
    { value: '16_7', label: 'iOS 16.7' },
  ],
  default: [
    { value: '14', label: 'Android 14' },
    { value: '13', label: 'Android 13' },
    { value: '12', label: 'Android 12' },
  ],
}

function getOSVersionOptions(platform?: string, deviceType = 'desktop') {
  if (deviceType === 'mobile') {
    return platform === 'macos' || platform === 'mac' ? MOBILE_OS_VERSION_OPTIONS.mac : MOBILE_OS_VERSION_OPTIONS.default
  }
  return OS_VERSION_OPTIONS[platform || 'windows'] ?? OS_VERSION_OPTIONS.windows
}

function defaultWebglRenderer(vendor?: string) {
  if (!vendor || vendor === 'random') return 'random'
  return (WEBGL_RENDERER_OPTIONS[vendor] ?? WEBGL_RENDERER_OPTIONS.Intel)[0]?.value || 'random'
}

const LANG_OPTIONS = [
  { value: 'ip', label: '基于访问 IP' },
  { value: 'zh-CN', label: '中文 (zh-CN)' },
  { value: 'en-US', label: 'English (en-US)' },
  { value: 'en-GB', label: 'English (en-GB)' },
  { value: 'es-ES', label: 'Español (es-ES)' },
  { value: 'ja-JP', label: '日本語 (ja-JP)' },
  { value: 'ko-KR', label: '한국어 (ko-KR)' },
  { value: 'fr-FR', label: 'Français (fr-FR)' },
  { value: 'de-DE', label: 'Deutsch (de-DE)' },
  { value: 'it-IT', label: 'Italiano (it-IT)' },
  { value: 'pt-BR', label: 'Português (pt-BR)' },
  { value: 'ru-RU', label: 'Русский (ru-RU)' },
]

const TIMEZONE_OPTIONS = [
  { value: 'ip', label: '基于访问 IP' },
  { value: 'system', label: '跟随系统时区' },
  // 亚洲
  { value: 'Asia/Shanghai', label: 'Asia/Shanghai (UTC+8)' },
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo (UTC+9)' },
  { value: 'Asia/Seoul', label: 'Asia/Seoul (UTC+9)' },
  { value: 'Asia/Singapore', label: 'Asia/Singapore (UTC+8)' },
  { value: 'Asia/Hong_Kong', label: 'Asia/Hong_Kong (UTC+8)' },
  { value: 'Asia/Dubai', label: 'Asia/Dubai (UTC+4)' },
  { value: 'Asia/Kolkata', label: 'Asia/Kolkata (UTC+5:30)' },
  // 美洲
  { value: 'America/New_York', label: 'America/New_York (UTC-5)' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (UTC-8)' },
  { value: 'America/Chicago', label: 'America/Chicago (UTC-6)' },
  { value: 'America/Denver', label: 'America/Denver (UTC-7)' },
  { value: 'America/Toronto', label: 'America/Toronto (UTC-5)' },
  { value: 'America/Sao_Paulo', label: 'America/Sao_Paulo (UTC-3)' },
  // EMEA
  { value: 'Europe/London', label: 'Europe/London (UTC+0)' },
  { value: 'Europe/Paris', label: 'Europe/Paris (UTC+1)' },
  { value: 'Europe/Berlin', label: 'Europe/Berlin (UTC+1)' },
  { value: 'Europe/Moscow', label: 'Europe/Moscow (UTC+3)' },
  // 大洋洲
  { value: 'Australia/Sydney', label: 'Australia/Sydney (UTC+10)' },
  { value: 'Pacific/Auckland', label: 'Pacific/Auckland (UTC+12)' },
]

const RESOLUTION_OPTIONS = [
  { value: RANDOM_OPTION_VALUE, label: '随机' },
  ...PRESET_RESOLUTIONS.map(r => ({ value: r, label: r })),
  { value: 'custom', label: '自定义...' },
]

const WEBGL_VENDOR_OPTIONS = [
  { value: 'random', label: '随机' },
  { value: 'Intel', label: 'Intel' },
  { value: 'NVIDIA', label: 'NVIDIA' },
  { value: 'AMD', label: 'AMD' },
  { value: 'Apple', label: 'Apple' },
]

const WEBGL_RENDERER_OPTIONS: Record<string, { value: string; label: string }[]> = {
  random: [
    { value: 'random', label: '随机' },
  ],
  Intel: [
    { value: 'random', label: '随机' },
    { value: 'Intel(R) UHD Graphics 630', label: 'UHD Graphics 630' },
    { value: 'Intel(R) UHD Graphics 620', label: 'UHD Graphics 620' },
    { value: 'Intel(R) HD Graphics 520', label: 'HD Graphics 520' },
    { value: 'Intel(R) Iris(R) Xe Graphics', label: 'Iris Xe Graphics' },
    { value: 'custom', label: '自定义...' },
  ],
  NVIDIA: [
    { value: 'random', label: '随机' },
    { value: 'NVIDIA GeForce RTX 3080', label: 'GeForce RTX 3080' },
    { value: 'NVIDIA GeForce RTX 3060', label: 'GeForce RTX 3060' },
    { value: 'NVIDIA GeForce GTX 1660', label: 'GeForce GTX 1660' },
    { value: 'NVIDIA GeForce GTX 1080 Ti', label: 'GeForce GTX 1080 Ti' },
    { value: 'custom', label: '自定义...' },
  ],
  AMD: [
    { value: 'random', label: '随机' },
    { value: 'AMD Radeon RX 6600', label: 'Radeon RX 6600' },
    { value: 'AMD Radeon RX 580', label: 'Radeon RX 580' },
    { value: 'AMD Radeon Vega 8', label: 'Radeon Vega 8' },
    { value: 'custom', label: '自定义...' },
  ],
  Apple: [
    { value: 'random', label: '随机' },
    { value: 'Apple M1', label: 'Apple M1' },
    { value: 'Apple M2', label: 'Apple M2' },
    { value: 'Apple M3', label: 'Apple M3' },
    { value: 'custom', label: '自定义...' },
  ],
}

const BOOL_OPTIONS = [
  { value: 'true', label: '启用' },
  { value: 'false', label: '禁用' },
]

const MODE_OPTIONS = [
  { value: 'random', label: '随机' },
  { value: 'real', label: '真实' },
  { value: 'disabled', label: '关闭' },
]

const CUSTOM_MODE_OPTIONS = [
  { value: 'random', label: '随机' },
  { value: 'custom', label: '自定义' },
  { value: 'disabled', label: '关闭' },
]

const GEOLOCATION_PERMISSION_OPTIONS = [
  { value: 'ask', label: '询问' },
  { value: 'allow', label: '允许' },
  { value: 'block', label: '禁用' },
]

const HARDWARE_CONCURRENCY_OPTIONS = [
  { value: RANDOM_OPTION_VALUE, label: '随机' },
  { value: '2', label: '2 核' },
  { value: '4', label: '4 核' },
  { value: '6', label: '6 核' },
  { value: '8', label: '8 核' },
  { value: '10', label: '10 核' },
  { value: '12', label: '12 核' },
  { value: '16', label: '16 核' },
]

const DEVICE_MEMORY_OPTIONS = [
  { value: RANDOM_OPTION_VALUE, label: '随机' },
  { value: '2', label: '2 GB' },
  { value: '4', label: '4 GB' },
  { value: '8', label: '8 GB' },
  { value: '16', label: '16 GB' },
  { value: '32', label: '32 GB' },
]

const COLOR_DEPTH_OPTIONS = [
  { value: RANDOM_OPTION_VALUE, label: '随机' },
  { value: '24', label: '24 位（标准）' },
  { value: '30', label: '30 位（HDR）' },
  { value: '32', label: '32 位' },
]

const WEBRTC_OPTIONS = [
  { value: 'default', label: '转发' },
  { value: 'disable_non_proxied_udp', label: '替换（推荐）' },
  { value: 'default_public_interface_only', label: '真实' },
  { value: 'disable_udp', label: '禁用 UDP' },
  { value: 'disabled', label: '禁用' },
  { value: 'default_public_and_private_interfaces', label: '公网+私网接口' },
]

const TOUCH_POINTS_OPTIONS = [
  { value: RANDOM_OPTION_VALUE, label: '随机' },
  { value: '0', label: '0（无触摸）' },
  { value: '1', label: '1 点触摸' },
  { value: '5', label: '5 点触摸' },
  { value: '10', label: '10 点触摸' },
]

const PRESET_OPTIONS = [
  { value: '', label: '选择预设...' },
  ...FINGERPRINT_PRESETS.map(p => ({ value: p.id, label: p.name })),
]

export function FingerprintPanel({ value, onChange }: FingerprintPanelProps) {
  const [config, setConfig] = useState<FingerprintConfig>(() => deserialize(value))
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [customRenderer, setCustomRenderer] = useState('')
  const [confirmSeedOpen, setConfirmSeedOpen] = useState(false)

  useEffect(() => {
    setConfig(deserialize(value))
  }, [value.join('\n')])

  const update = (patch: Partial<FingerprintConfig>) => {
    const next = { ...config, ...patch }
    setConfig(next)
    onChange(serialize(next))
  }

  const updateUA = (patch: Partial<FingerprintConfig>) => {
    const next = { ...config, ...patch }
    const withUserAgent = { ...next, userAgent: buildUserAgent(next) }
    setConfig(withUserAgent)
    onChange(serialize(withUserAgent))
  }

  const randomizeScreenHardware = (patch: Partial<FingerprintConfig> = {}) => {
    update({
      ...randomScreenHardwarePatch(),
      ...patch,
    })
  }

  const handlePresetChange = (presetId: string) => {
    if (!presetId) return
    const preset = FINGERPRINT_PRESETS.find(p => p.id === presetId)
    if (!preset) return
    // 应用预设时自动生成新种子，保留未知参数
    const next: FingerprintConfig = {
      ...preset.config,
      seed: randomFingerprintSeed(),
      unknownArgs: config.unknownArgs,
    }
    next.userAgent = buildUserAgent(next)
    setConfig(next)
    onChange(serialize(next))
  }

  const handleAdvancedChange = (text: string) => {
    const args = text.split('\n').map(s => s.trim()).filter(Boolean)
    const parsed = deserialize(args)
    setConfig(parsed)
    onChange(serialize(parsed))
  }

  const rendererOptions = config.webglVendor
    ? (WEBGL_RENDERER_OPTIONS[config.webglVendor] ?? [{ value: 'random', label: '随机' }, { value: 'custom', label: '自定义...' }])
    : WEBGL_RENDERER_OPTIONS.random

  const isCustomRenderer = config.webglRenderer
    ? config.webglRenderer === 'custom' || !rendererOptions.some(o => o.value === config.webglRenderer && o.value !== 'custom')
    : false

  const advancedText = serialize(config).join('\n')
  const uaOSOptions = getOSVersionOptions(config.platform, config.deviceType || 'desktop')

  return (
    <div className="space-y-4">
      {/* 指纹种子 */}
      <div className="p-3 rounded-lg bg-[var(--color-bg-hover)] border border-[var(--color-border)] space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wide">指纹种子（Fingerprint Seed）</span>
          <span className="text-xs text-[var(--color-text-muted)]">决定所有随机噪声的根值，不同种子 = 不同指纹</span>
        </div>
        <div className="flex items-center gap-2">
          <Input
            value={config.seed ?? ''}
            onChange={e => update({ seed: e.target.value || undefined })}
            placeholder="留空则由系统按 ProfileId 自动生成"
            className="flex-1 font-mono text-sm"
          />
          <button
            type="button"
            title="随机生成新种子"
            onClick={() => {
              if (config.seed) {
                setConfirmSeedOpen(true)
              } else {
                update({ seed: randomFingerprintSeed() })
              }
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs bg-[var(--color-primary)] text-white hover:opacity-90 transition-opacity shrink-0"
          >
            <RefreshCw className="w-3.5 h-3.5" />
            随机
          </button>
        </div>
      </div>

      <ConfirmModal
        open={confirmSeedOpen}
        onClose={() => setConfirmSeedOpen(false)}
        onConfirm={() => update({ seed: randomFingerprintSeed() })}
        title="重新生成指纹种子"
        content="重新生成后，当前指纹将完全改变，浏览器的 Canvas、WebGL、Audio 等所有噪声特征都会随之变化。确定继续？"
        confirmText="确定重新生成"
        danger
      />

      {/* 预设选择 */}
      <div className="flex items-center gap-3 p-3 rounded-lg bg-[var(--color-bg-hover)] border border-[var(--color-border)]">
        <Wand2 className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
        <div className="flex-1 min-w-0">
          <Select
            value=""
            onChange={e => handlePresetChange(e.target.value)}
            options={PRESET_OPTIONS}
          />
        </div>
        <span className="text-xs text-[var(--color-text-muted)] shrink-0">选择后覆盖当前配置</span>
      </div>

      {/* 基础身份 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">基础身份</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="设备类型">
            <Select
              value={config.deviceType ?? 'desktop'}
              onChange={e => updateUA({ deviceType: e.target.value || 'desktop', osVersion: defaultOSVersion(config.platform, e.target.value || 'desktop') })}
              options={DEVICE_TYPE_OPTIONS}
            />
          </FormItem>
          <FormItem label="浏览器品牌">
            <Select value={config.brand ?? 'Chrome'} onChange={e => updateUA({ brand: e.target.value || 'Chrome' })} options={BRAND_OPTIONS} />
          </FormItem>
          <FormItem label="操作系统">
            <Select
              value={config.platform ?? 'windows'}
              onChange={e => updateUA({ platform: e.target.value || 'windows', osVersion: defaultOSVersion(e.target.value, config.deviceType || 'desktop') })}
              options={PLATFORM_OPTIONS}
            />
          </FormItem>
          <FormItem label="系统版本">
            <Select
              value={config.osVersion ?? defaultOSVersion(config.platform, config.deviceType || 'desktop')}
              onChange={e => updateUA({ osVersion: e.target.value || undefined })}
              options={uaOSOptions}
            />
          </FormItem>
          <FormItem label="浏览器大版本">
            <Select value={config.browserMajor ?? '139'} onChange={e => updateUA({ browserMajor: e.target.value || '139' })} options={BROWSER_MAJOR_OPTIONS} />
          </FormItem>
          <FormItem label="语言">
            <Select value={config.lang ?? 'ip'} onChange={e => update({ lang: e.target.value || 'ip' })} options={LANG_OPTIONS} />
          </FormItem>
          <FormItem label="时区">
            <Select value={config.timezone ?? 'ip'} onChange={e => update({ timezone: e.target.value || 'ip' })} options={TIMEZONE_OPTIONS.map(opt =>
              opt.value === 'system'
                ? { ...opt, label: `跟随系统时区 (当前: ${getSystemTimezone()})` }
                : opt
            )} />
          </FormItem>
        </div>
        <div className="mt-4">
          <FormItem label="User Agent">
            <Textarea
              value={config.userAgent ?? buildUserAgent(config)}
              onChange={e => update({ userAgent: e.target.value || undefined })}
              rows={2}
              placeholder="Mozilla/5.0 ..."
            />
          </FormItem>
        </div>
      </div>

      {/* 屏幕与硬件 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">屏幕与硬件</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="分辨率">
            <Select
              value={config.resolution ?? '1920,1080'}
              onChange={e => {
                if (e.target.value === RANDOM_OPTION_VALUE) {
                  randomizeScreenHardware()
                } else {
                  update({ resolution: e.target.value || '1920,1080', customResolution: undefined })
                }
              }}
              options={RESOLUTION_OPTIONS}
            />
          </FormItem>
          {config.resolution === 'custom' && (
            <FormItem label="自定义分辨率">
              <Input value={config.customResolution ?? ''} onChange={e => update({ customResolution: e.target.value || undefined })} placeholder="1600,900" />
            </FormItem>
          )}
          <FormItem label="色深">
            <Select
              value={config.colorDepth ?? '24'}
              onChange={e => {
                if (e.target.value === RANDOM_OPTION_VALUE) {
                  randomizeScreenHardware()
                } else {
                  update({ colorDepth: e.target.value || '24' })
                }
              }}
              options={COLOR_DEPTH_OPTIONS}
            />
          </FormItem>
          <FormItem label="CPU 核心数">
            <Select
              value={config.hardwareConcurrency ?? '8'}
              onChange={e => {
                if (e.target.value === RANDOM_OPTION_VALUE) {
                  randomizeScreenHardware()
                } else {
                  update({ hardwareConcurrency: e.target.value || '8' })
                }
              }}
              options={HARDWARE_CONCURRENCY_OPTIONS}
            />
          </FormItem>
          <FormItem label="设备内存">
            <Select
              value={config.deviceMemory ?? '8'}
              onChange={e => {
                if (e.target.value === RANDOM_OPTION_VALUE) {
                  randomizeScreenHardware()
                } else {
                  update({ deviceMemory: e.target.value || '8' })
                }
              }}
              options={DEVICE_MEMORY_OPTIONS}
            />
          </FormItem>
          <FormItem label="触摸点数">
            <Select
              value={config.touchPoints ?? '0'}
              onChange={e => {
                if (e.target.value === RANDOM_OPTION_VALUE) {
                  randomizeScreenHardware()
                } else {
                  update({ touchPoints: e.target.value || '0' })
                }
              }}
              options={TOUCH_POINTS_OPTIONS}
            />
          </FormItem>
        </div>
      </div>

      {/* 渲染指纹 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">渲染指纹</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="WebGL 供应商">
            <Select
              value={config.webglVendor ?? 'random'}
              onChange={e => {
                const vendor = e.target.value || 'random'
                update({ webglVendor: vendor, webglRenderer: defaultWebglRenderer(vendor) })
              }}
              options={WEBGL_VENDOR_OPTIONS}
            />
          </FormItem>
          <FormItem label="WebGL 渲染器">
            {isCustomRenderer ? (
              <Input
                value={config.webglRenderer === 'custom' ? customRenderer : config.webglRenderer ?? ''}
                onChange={e => {
                  setCustomRenderer(e.target.value)
                  update({ webglRenderer: e.target.value || 'custom' })
                }}
                placeholder="自定义渲染器名称"
              />
            ) : (
              <Select
                value={config.webglRenderer ?? defaultWebglRenderer(config.webglVendor)}
                onChange={e => {
                  if (e.target.value === 'custom') {
                    setCustomRenderer('')
                    update({ webglRenderer: 'custom' })
                  } else {
                    update({ webglRenderer: e.target.value || defaultWebglRenderer(config.webglVendor) })
                  }
                }}
                options={rendererOptions}
                disabled={!config.webglVendor}
              />
            )}
          </FormItem>
          <FormItem label="Canvas 噪声">
            <Select
              value={config.canvasNoise === undefined ? 'true' : String(config.canvasNoise)}
              onChange={e => update({ canvasNoise: e.target.value !== 'false' })}
              options={[
                { value: 'true', label: '随机' },
                { value: 'false', label: '关闭' },
              ]}
            />
          </FormItem>
          <FormItem label="WebGL 图像">
            <Select
              value={config.webglImageMode ?? 'random'}
              onChange={e => update({ webglImageMode: e.target.value || 'random' })}
              options={CUSTOM_MODE_OPTIONS}
            />
          </FormItem>
          <FormItem label="WebGL 元数据">
            <Select
              value={config.webglMetadataMode ?? 'random'}
              onChange={e => update({ webglMetadataMode: e.target.value || 'random' })}
              options={CUSTOM_MODE_OPTIONS}
            />
          </FormItem>
          <FormItem label="WebGPU">
            <Select
              value={config.webgpuMode ?? 'webgl'}
              onChange={e => update({ webgpuMode: e.target.value || 'webgl' })}
              options={[
                { value: 'webgl', label: '基于 WebGL' },
                { value: 'real', label: '真实' },
                { value: 'disabled', label: '禁用' },
              ]}
            />
          </FormItem>
          <FormItem label="Audio 噪声">
            <Select
              value={config.audioNoise === undefined ? 'true' : String(config.audioNoise)}
              onChange={e => update({ audioNoise: e.target.value !== 'false' })}
              options={[
                { value: 'true', label: '随机' },
                { value: 'false', label: '关闭' },
              ]}
            />
          </FormItem>
          <FormItem label="ClientRects">
            <Select
              value={config.clientRectsMode ?? 'random'}
              onChange={e => update({ clientRectsMode: e.target.value || 'random' })}
              options={MODE_OPTIONS}
            />
          </FormItem>
          <FormItem label="SpeechVoices">
            <Select
              value={config.speechVoicesMode ?? 'random'}
              onChange={e => update({ speechVoicesMode: e.target.value || 'random' })}
              options={MODE_OPTIONS}
            />
          </FormItem>
          <FormItem label="设备名称">
            <Select
              value={config.deviceNameMode ?? 'random'}
              onChange={e => update({ deviceNameMode: e.target.value || 'random' })}
              options={CUSTOM_MODE_OPTIONS}
            />
          </FormItem>
        </div>
      </div>

      {/* 网络与隐私 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">网络与隐私</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="WebRTC 策略">
            <Select value={config.webrtcPolicy ?? 'disable_non_proxied_udp'} onChange={e => update({ webrtcPolicy: e.target.value || 'disable_non_proxied_udp' })} options={WEBRTC_OPTIONS} />
          </FormItem>
          <FormItem label="Do Not Track">
            <Select
              value={config.doNotTrack === undefined ? 'false' : String(config.doNotTrack)}
              onChange={e => update({ doNotTrack: e.target.value === 'true' })}
              options={[
                { value: 'true', label: '开启' },
                { value: 'false', label: '关闭' },
              ]}
            />
          </FormItem>
          <FormItem label="地理位置权限">
            <Select
              value={config.geolocationPermission ?? 'ask'}
              onChange={e => update({ geolocationPermission: e.target.value || 'ask' })}
              options={GEOLOCATION_PERMISSION_OPTIONS}
            />
          </FormItem>
          <FormItem label="地理位置基于 IP">
            <Select
              value={config.geolocationBasedOnIp === undefined ? 'true' : String(config.geolocationBasedOnIp)}
              onChange={e => update({ geolocationBasedOnIp: e.target.value !== 'false' })}
              options={BOOL_OPTIONS}
            />
          </FormItem>
          <FormItem label="端口扫描保护">
            <Select
              value={config.portScanProtection === undefined ? 'false' : String(config.portScanProtection)}
              onChange={e => update({ portScanProtection: e.target.value === 'true' })}
              options={BOOL_OPTIONS}
            />
          </FormItem>
          <FormItem label="Cloudflare 验证优化">
            <Select
              value={config.cloudflareOptimize === undefined ? 'false' : String(config.cloudflareOptimize)}
              onChange={e => update({ cloudflareOptimize: e.target.value === 'true' })}
              options={BOOL_OPTIONS}
            />
          </FormItem>
          <FormItem label="媒体设备 (摄像头,麦克风,扬声器)">
            <Input
              value={config.mediaDevices ?? '2,1,1'}
              onChange={e => update({ mediaDevices: e.target.value || '2,1,1' })}
              placeholder="2,1,1"
            />
          </FormItem>
        </div>
      </div>

      {/* 字体 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">字体</p>
        <FormItem label="字体列表">
          <Input
            value={config.fonts ?? 'Arial,Helvetica,Times New Roman,Courier New,Verdana'}
            onChange={e => update({ fonts: e.target.value || 'Arial,Helvetica,Times New Roman,Courier New,Verdana' })}
            placeholder="Arial,Helvetica,Times New Roman（逗号分隔）"
          />
        </FormItem>
      </div>

      {/* 高级模式 */}
      <div className="border border-[var(--color-border)] rounded-lg overflow-hidden">
        <button
          type="button"
          className="w-full flex items-center justify-between px-4 py-2.5 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-bg-hover)] transition-colors"
          onClick={() => setAdvancedOpen(v => !v)}
        >
          <span>高级模式（原始参数）</span>
          {advancedOpen ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </button>
        {advancedOpen && (
          <div className="px-4 pb-4 pt-2 border-t border-[var(--color-border)]">
            <p className="text-xs text-[var(--color-text-muted)] mb-2">每行一个参数，修改后自动同步到上方控件</p>
            <Textarea
              value={advancedText}
              onChange={e => handleAdvancedChange(e.target.value)}
              rows={6}
              placeholder="--fingerprint-brand=Chrome"
            />
          </div>
        )}
      </div>
    </div>
  )
}
