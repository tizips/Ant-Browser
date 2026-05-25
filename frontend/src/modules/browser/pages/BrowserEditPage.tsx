import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FolderOpen, Layers } from 'lucide-react'
import { Button, Card, ConfirmModal, FormItem, Input, Modal, Select, Textarea, toast } from '../../../shared/components'
import type { BrowserCore, BrowserProfileInput, BrowserProxy, BrowserGroup } from '../types'
import { createBrowserProfile, fetchAllTags, fetchBrowserCores, fetchBrowserProfiles, fetchBrowserProxies, fetchBrowserSettings, fetchGroups, openUserDataDir, updateBrowserProfile } from '../api'
import { FingerprintPanel } from '../components/FingerprintPanel'
import { TagInput } from '../components/TagInput'
import { GroupSelector } from '../components/GroupSelector'
import { ProxyPickerModal } from '../components/ProxyPickerModal'
import { createRandomizedFingerprintConfig, deserialize, serialize } from '../utils/fingerprintSerializer'

const fallbackLowLaunchArgs = ['--disable-sync', '--no-first-run']
const directProxyID = '__direct__'
const defaultIconColor = '#2563EB'
const platformOptions = [
  { value: 'none', label: '无平台', name: '', url: '' },
  { value: 'google', label: 'Google', name: 'Google', url: 'https://accounts.google.com/' },
  { value: 'facebook', label: 'Facebook', name: 'Facebook', url: 'https://www.facebook.com/' },
  { value: 'tiktok', label: 'TikTok', name: 'TikTok', url: 'https://www.tiktok.com/login' },
  { value: 'amazon', label: 'Amazon', name: 'Amazon', url: 'https://www.amazon.com/' },
  { value: 'paypal', label: 'PayPal', name: 'PayPal', url: 'https://www.paypal.com/signin' },
  { value: 'custom', label: '自定义平台', name: '自定义平台', url: '' },
]

function normalizeLaunchArgs(args: string[]): string[] {
  return (args || []).map(item => item.trim()).filter(Boolean)
}

function resolveDefaultLaunchArgs(args: string[]): string[] {
  const normalized = normalizeLaunchArgs(args)
  return normalized.length > 0 ? normalized : fallbackLowLaunchArgs
}

function resolvePoolProxySelection(
  proxyId: string,
  proxyConfig: string,
  proxies: BrowserProxy[],
): { proxyId: string; proxyConfig: string } {
  const normalizedProxyId = proxyId.trim()
  if (normalizedProxyId) {
    const matchedByID = proxies.find((proxy) => proxy.proxyId.trim() === normalizedProxyId)
    if (matchedByID?.proxyId) {
      return { proxyId: matchedByID.proxyId, proxyConfig: '' }
    }
  }

  const rawProxyConfig = proxyConfig.trim()
  const normalizedConfig = rawProxyConfig.toLowerCase()
  if (normalizedConfig) {
    const matchedByConfig = proxies.find((proxy) => (proxy.proxyConfig || '').trim().toLowerCase() === normalizedConfig)
    if (matchedByConfig?.proxyId) {
      return { proxyId: matchedByConfig.proxyId, proxyConfig: '' }
    }
    return { proxyId: '', proxyConfig: rawProxyConfig }
  }

  const directProxy = proxies.find((proxy) => proxy.proxyId === directProxyID)
  return { proxyId: directProxy?.proxyId || '', proxyConfig: '' }
}

export function BrowserEditPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isCreate = id === 'new'
  const [formData, setFormData] = useState<BrowserProfileInput>({
    profileName: '',
    username: '',
    password: '',
    platform: 'none',
    platformName: '',
    platformUrl: '',
    userDataDir: '',
    coreId: '',
    fingerprintArgs: [],
    proxyId: directProxyID,
    proxyConfig: '',
    launchArgs: [],
    tags: [],
    keywords: [],
    twoFaSecret: '',
    iconColor: '',
    groupId: '',
  })
  const [cores, setCores] = useState<BrowserCore[]>([])
  const [proxies, setProxies] = useState<BrowserProxy[]>([])
  const [groups, setGroups] = useState<BrowserGroup[]>([])
  const [launchArgsText, setLaunchArgsText] = useState('')
  const [allTags, setAllTags] = useState<string[]>([])
  const [saving, setSaving] = useState(false)
  const [proxyPickerOpen, setProxyPickerOpen] = useState(false)
  const [isDirty, setIsDirty] = useState(false)
  const [leaveConfirm, setLeaveConfirm] = useState(false)
  const [saveError, setSaveError] = useState('')

  useEffect(() => {
    const loadData = async () => {
      const [coreList, proxyList, tagList, groupList, settings] = await Promise.all([
        fetchBrowserCores(),
        fetchBrowserProxies(),
        fetchAllTags(),
        fetchGroups(),
        fetchBrowserSettings(),
      ])
      const resolvedDefaultLaunchArgs = resolveDefaultLaunchArgs(settings.defaultLaunchArgs || [])
      setCores(coreList)
      setProxies(proxyList)
      setAllTags(tagList)
      setGroups(groupList)

      if (isCreate) {
        const resolved = resolvePoolProxySelection('', '', proxyList)
        const randomizedFingerprintArgs = serialize(createRandomizedFingerprintConfig(deserialize(settings.defaultFingerprintArgs || [])))
        setFormData((prev) => ({
          ...prev,
          proxyId: resolved.proxyId || directProxyID,
          proxyConfig: '',
          fingerprintArgs: randomizedFingerprintArgs,
        }))
        setLaunchArgsText(resolvedDefaultLaunchArgs.join('\n'))
        return
      }
      const list = await fetchBrowserProfiles()
      const current = list.find(item => item.profileId === id)
      if (!current) return
      const currentLaunchArgs = normalizeLaunchArgs(current.launchArgs)
      const normalizedCoreId = !current.coreId || current.coreId.toLowerCase() === 'default'
        ? ''
        : current.coreId
      const resolvedProxy = resolvePoolProxySelection(current.proxyId || '', current.proxyConfig || '', proxyList)
      setFormData({
        profileName: current.profileName,
        username: current.username || current.profileName,
        password: current.password || '',
        platform: current.platform || 'none',
        platformName: current.platformName || '',
        platformUrl: current.platformUrl || '',
        userDataDir: current.userDataDir,
        coreId: normalizedCoreId,
        fingerprintArgs: current.fingerprintArgs,
        proxyId: resolvedProxy.proxyId,
        proxyConfig: resolvedProxy.proxyConfig,
        launchArgs: currentLaunchArgs,
        tags: current.tags,
        keywords: current.keywords || [],
        twoFaSecret: current.twoFaSecret || '',
        iconColor: current.iconColor || '',
        groupId: current.groupId || '',
      })
      setLaunchArgsText(currentLaunchArgs.join('\n'))
    }
    loadData()
  }, [id, isCreate])

  const handleChange = (field: keyof BrowserProfileInput, value: string | string[]) => {
    setIsDirty(true)
    setFormData(prev => {
      if (field === 'proxyId') {
        return { ...prev, proxyId: typeof value === 'string' ? value : '', proxyConfig: '' }
      }
      if (field === 'profileName' && typeof value === 'string' && !prev.username) {
        return { ...prev, profileName: value, username: value }
      }
      return { ...prev, [field]: value }
    })
  }

  const handlePlatformChange = (platform: string) => {
    setIsDirty(true)
    const preset = platformOptions.find(item => item.value === platform) || platformOptions[0]
    setFormData(prev => ({
      ...prev,
      platform,
      platformName: preset.name,
      platformUrl: preset.url,
    }))
  }

  const handleSave = async () => {
    setSaving(true)
    const resolvedProxyId = (formData.proxyId || '').trim()
    const resolvedProxyConfig = (formData.proxyConfig || '').trim()
    const payload: BrowserProfileInput = {
      ...formData,
      username: (formData.username || '').trim(),
      password: (formData.password || '').trim(),
      platform: (formData.platform || 'none').trim(),
      platformName: (formData.platformName || '').trim(),
      platformUrl: (formData.platformUrl || '').trim(),
      proxyId: resolvedProxyId,
      proxyConfig: '',
      launchArgs: normalizeLaunchArgs(launchArgsText.split('\n')),
    }
    if (payload.platform === 'none') {
      payload.platformName = ''
      payload.platformUrl = ''
    }
    if (!resolvedProxyId) {
      if (resolvedProxyConfig) {
        payload.proxyConfig = resolvedProxyConfig
      } else {
        payload.proxyId = directProxyID
      }
    }
    try {
      if (isCreate) {
        await createBrowserProfile(payload)
        toast.success('配置已创建')
      } else if (id) {
        await updateBrowserProfile(id, payload)
        toast.success('配置已更新')
      }
      setIsDirty(false)
      navigate('/browser/list')
    } catch (error: any) {
      setSaveError(typeof error === 'string' ? error : error?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleBack = () => {
    if (isDirty) { setLeaveConfirm(true) } else { navigate('/browser/list') }
  }

  const defaultCore = cores.find(c => c.isDefault)

  const handleOpenUserDataDir = async () => {
    if (!formData.userDataDir.trim()) {
      toast.error('请先输入用户数据目录')
      return
    }
    try {
      await openUserDataDir(formData.userDataDir)
    } catch (error: unknown) {
      toast.error((error as Error)?.message || '打开目录失败')
    }
  }

  const handleProxyListUpdated = (nextProxies: BrowserProxy[]) => {
    setProxies(nextProxies)
  }

  const handleProxyDeleted = (deletedProxyId: string, nextProxies: BrowserProxy[]) => {
    setProxies(nextProxies)
    if (formData.proxyId !== deletedProxyId) {
      return
    }

    const fallbackProxy = nextProxies.find((proxy) => proxy.proxyId === directProxyID)
    if (fallbackProxy) {
      handleChange('proxyId', fallbackProxy.proxyId)
      return
    }

    handleChange('proxyId', '')
  }

  return (
    <div className="space-y-5 animate-fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">{isCreate ? '新建配置' : '编辑配置'}</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">完善指纹与启动参数</p>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" size="sm" onClick={handleBack}>返回列表</Button>
          <Button size="sm" onClick={handleSave} loading={saving}>保存配置</Button>
        </div>
      </div>

      <Card title="基础信息" subtitle="实例与配置名称">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="配置名称" required>
            <Input value={formData.profileName} onChange={e => handleChange('profileName', e.target.value)} placeholder="请输入配置名称" />
          </FormItem>
          <FormItem label="分组">
            <GroupSelector
              groups={groups}
              value={formData.groupId || ''}
              onChange={groupId => handleChange('groupId', groupId)}
              placeholder="未分组"
              className="w-full"
            />
          </FormItem>
          <FormItem label="平台">
            <Select
              value={formData.platform || 'none'}
              onChange={e => handlePlatformChange(e.target.value)}
              options={platformOptions.map(item => ({ value: item.value, label: item.label }))}
            />
          </FormItem>
          {(formData.platform || 'none') !== 'none' && (
            <>
              <FormItem label="平台名称">
                <Input
                  value={formData.platformName || ''}
                  onChange={e => handleChange('platformName', e.target.value)}
                  placeholder="例如 Google"
                />
              </FormItem>
              <FormItem label="平台链接">
                <Input
                  value={formData.platformUrl || ''}
                  onChange={e => handleChange('platformUrl', e.target.value)}
                  placeholder="https://accounts.google.com/"
                />
              </FormItem>
            </>
          )}
          <FormItem label="用户名">
            <Input
              value={formData.username || ''}
              onChange={e => handleChange('username', e.target.value)}
              placeholder="平台用户名，默认使用配置名称"
            />
          </FormItem>
          <FormItem label="密码">
            <Input
              type="password"
              value={formData.password || ''}
              onChange={e => handleChange('password', e.target.value)}
              placeholder="平台密码，平台页可快捷输入"
            />
          </FormItem>
          <FormItem label="2FA 密钥">
            <Input
              value={formData.twoFaSecret || ''}
              onChange={e => handleChange('twoFaSecret', e.target.value)}
              placeholder="Google Authenticator Base32 密钥或 otpauth:// 链接"
            />
          </FormItem>
          <FormItem label="用户数据目录（留空自动生成）">
            <div className="flex gap-2">
              <Input
                value={formData.userDataDir}
                onChange={e => handleChange('userDataDir', e.target.value)}
                placeholder="留空自动生成"
                className="flex-1"
              />
              <Button variant="secondary" size="sm" onClick={handleOpenUserDataDir} title="在资源管理器中打开">
                <FolderOpen className="w-4 h-4" />
              </Button>
            </div>
          </FormItem>
          <FormItem label="内核">
            <Select
              value={formData.coreId}
              onChange={e => handleChange('coreId', e.target.value)}
              options={
                cores.length > 0 ? [
                  { value: '', label: defaultCore ? `使用默认 (${defaultCore.coreName})` : '使用默认内核' },
                  ...cores.map(c => ({ value: c.coreId, label: c.coreName })),
                ] : [
                  { value: '', label: '暂无内核，请添加内核' }
                ]
              }
            />
          </FormItem>
          <FormItem label="标签">
            <TagInput
              value={formData.tags}
              onChange={tags => handleChange('tags', tags)}
              suggestions={allTags}
              placeholder="输入标签后按回车，支持从已有标签选择"
            />
          </FormItem>
          <FormItem label="实例图标颜色">
            <div className="flex gap-2">
              <input
                type="color"
                value={formData.iconColor || defaultIconColor}
                onChange={e => handleChange('iconColor', e.target.value)}
                className="h-10 w-12 rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-1"
                title="选择实例 Dock 图标纯色"
              />
              <Input
                value={formData.iconColor || ''}
                onChange={e => handleChange('iconColor', e.target.value)}
                placeholder="留空新建时随机，例如 #2563EB"
                className="flex-1"
              />
            </div>
          </FormItem>
        </div>
      </Card>

      <Card title="代理配置" subtitle="仅支持从代理池选择（包含直连节点）">
        <div className="grid grid-cols-1 gap-4">
          <FormItem label="代理池选择">
            <div className="flex gap-2">
              <Select
                value={formData.proxyId}
                onChange={e => handleChange('proxyId', e.target.value)}
                options={
                  proxies.length > 0 ? [
                    ...(formData.proxyId === '' && formData.proxyConfig
                      ? [{ value: '', label: '接口自定义代理（保持原值）' }]
                      : []),
                    ...proxies.map(p => ({ value: p.proxyId, label: p.proxyName || p.proxyId })),
                  ] : [{ value: '', label: '暂无代理，请先到代理池创建' }]
                }
                className="flex-1"
              />
              <Button variant="secondary" size="sm" onClick={() => setProxyPickerOpen(true)} title="按分组选择代理">
                <Layers className="w-4 h-4" />
              </Button>
            </div>
          </FormItem>
        </div>
        <p className="text-xs text-[var(--color-text-muted)] mt-2">
          已移除手动代理输入，实例默认按代理池节点生效。
          {formData.proxyId === '' && formData.proxyConfig ? ' 当前实例为接口自定义代理，未改动代理选择时会保持原值。' : ''}
        </p>
      </Card>

      <ProxyPickerModal
        open={proxyPickerOpen}
        currentProxyId={formData.proxyId}
        onSelect={proxy => handleChange('proxyId', proxy.proxyId)}
        onProxyListUpdated={handleProxyListUpdated}
        onProxyDeleted={handleProxyDeleted}
        onClose={() => setProxyPickerOpen(false)}
      />

      <Card title="指纹配置" subtitle="配置浏览器指纹参数">
        <FingerprintPanel
          value={formData.fingerprintArgs}
          onChange={args => handleChange('fingerprintArgs', args)}
        />
      </Card>

      <Card title="启动参数" subtitle={isCreate ? '新建时默认填入轻量参数模板，直接改这里即可' : '每行一个参数'}>
        <div className="space-y-2">
          <Textarea
            value={launchArgsText}
            onChange={e => { setLaunchArgsText(e.target.value); setIsDirty(true) }}
            rows={6}
            placeholder="--disable-sync"
          />
          {isCreate && (
            <p className="text-xs text-[var(--color-text-muted)]">这里默认就是轻量参数模板；需要更复杂的参数，直接在此基础上修改。</p>
          )}
        </div>
      </Card>

      <ConfirmModal
        open={leaveConfirm}
        onClose={() => setLeaveConfirm(false)}
        onConfirm={() => navigate('/browser/list')}
        title="放弃未保存的更改？"
        content="当前页面有未保存的修改，离开后将丢失这些更改。"
        confirmText="放弃并离开"
        cancelText="继续编辑"
        danger
      />

      <Modal
        open={!!saveError}
        onClose={() => setSaveError('')}
        title="保存失败"
        width="420px"
        footer={<Button onClick={() => setSaveError('')}>知道了</Button>}
      >
        <div className="text-[var(--color-text-secondary)]">{saveError}</div>
      </Modal>
    </div>
  )
}
