import { Link } from 'react-router-dom'
import { ChevronLeft, ChevronRight, Copy, Key, Play, RotateCcw, Settings, Square, Trash2 } from 'lucide-react'

import { Badge, Button, Card, Table } from '../../../shared/components'
import type { SortOrder, SorterResult, TableColumn } from '../../../shared/components/Table'

import type { BrowserCore, BrowserProfile, BrowserProxy } from '../types'
import type { BrowserViewMode } from './BrowserListLayout'
import { KeywordInlineRow, LaunchCodeCell } from './BrowserListWidgets'

type ProfileStatusVariant = 'default' | 'success' | 'error' | 'warning' | 'info'

interface ProfileStatus {
  variant: ProfileStatusVariant
  label: string
}

interface BrowserProfilesPanelProps {
  loading: boolean
  viewMode: BrowserViewMode
  profiles: BrowserProfile[]
  totalCount: number
  proxies: BrowserProxy[]
  selectedIds: Set<string>
  sortColumn: string
  sortOrder: SortOrder
  currentPage: number
  pageSize: number
  resolveProfileCore: (profile: BrowserProfile) => BrowserCore | null
  getProfileCoreLabel: (profile: BrowserProfile) => string
  getProfileStatus: (profile: BrowserProfile) => ProfileStatus
  isProfileStarting: (profileId: string) => boolean
  isProfileStopping: (profileId: string) => boolean
  isProfileBusy: (profileId: string) => boolean
  onToggleSelect: (profileId: string) => void
  onSelectAll: () => void
  onDeselectAll: () => void
  onSortChange: (sorter: SorterResult) => void
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  onRefreshProfiles: () => void
  onStart: (profileId: string) => void
  onStop: (profileId: string) => void
  onRestart: (profileId: string) => void
  onOpenKeywords: (profile: BrowserProfile) => void
  onOpenCopy: (profile: BrowserProfile) => void
  onDelete: (profileId: string) => void
}

const formatTime = (value?: string) => {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString('zh-CN')
}

const shortProfileId = (value: string) => {
  if (!value) return '-'
  if (value.length <= 12) return value
  return `${value.slice(0, 8)}...${value.slice(-4)}`
}

function formatProxyLabel(profile: BrowserProfile, proxy?: BrowserProxy): string {
  if (proxy?.proxyName) {
    return proxy.proxyName
  }
  if (profile.proxyId) {
    return profile.proxyId
  }
  const customProxy = (profile.proxyConfig || '').trim()
  if (customProxy) {
    return `自定义: ${customProxy}`
  }
  return '-'
}

function BrowserProfileCard({
  profile,
  proxy,
  isSelected,
  status,
  coreLabel,
  isStarting,
  isStopping,
  isBusy,
  onToggleSelect,
  onRefreshProfiles,
  onStart,
  onStop,
  onRestart,
  onOpenKeywords,
  onOpenCopy,
  onDelete,
}: {
  profile: BrowserProfile
  proxy: BrowserProxy | undefined
  isSelected: boolean
  status: ProfileStatus
  coreLabel: string
  isStarting: boolean
  isStopping: boolean
  isBusy: boolean
  onToggleSelect: (profileId: string) => void
  onRefreshProfiles: () => void
  onStart: (profileId: string) => void
  onStop: (profileId: string) => void
  onRestart: (profileId: string) => void
  onOpenKeywords: (profile: BrowserProfile) => void
  onOpenCopy: (profile: BrowserProfile) => void
  onDelete: (profileId: string) => void
}) {
  return (
    <div
      className={`flex flex-col border rounded-xl bg-[var(--color-bg-surface)] p-3 shadow-[0_1px_4px_rgba(0,0,0,0.08)] transition-all duration-200 h-[320px] overflow-hidden
        ${isSelected ? 'border-[var(--color-accent)] ring-1 ring-[var(--color-accent)]/20' : 'border-[var(--color-border-default)] hover:border-[var(--color-accent)]'}
      `}
    >
      <div className="flex flex-col gap-3 pb-3 border-b border-[var(--color-border-muted)]/50 shrink-0">
        <div className="flex justify-between items-start gap-2">
          <div className="flex items-center gap-2 flex-wrap">
            <input
              type="checkbox"
              className="w-4 h-4 rounded cursor-pointer accent-[var(--color-accent)] mt-0.5 shrink-0"
              checked={isSelected}
              onChange={() => onToggleSelect(profile.profileId)}
            />
            <Link className="text-[var(--color-accent)] font-medium text-sm hover:text-[var(--color-accent)] transition-colors truncate max-w-[200px]" to={`/browser/detail/${profile.profileId}`}>
              {profile.profileName}
            </Link>
            <span
              className="max-w-[180px] truncate rounded bg-[var(--color-bg-muted)] px-2 py-0.5 text-xs font-medium text-[var(--color-text-muted)]"
              title={profile.profileId}
            >
              序号 {shortProfileId(profile.profileId)}
            </span>
            {profile.tags && profile.tags.length > 0 && (
              <div className="flex gap-1 ml-1">
                {profile.tags.map(tag => <Badge variant="default" key={tag}>{tag}</Badge>)}
              </div>
            )}
          </div>

          <Badge variant={status.variant} dot dotClassName="w-2 h-2 shrink-0">
            {status.label}
          </Badge>
        </div>

        <div className="flex items-center gap-1 flex-wrap">
          {profile.running ? (
            <Button size="sm" variant="secondary" onClick={() => onStop(profile.profileId)} title={isStopping ? '停止中' : '停止'} loading={isStopping}>
              {!isStopping && <Square className="w-4 h-4 mr-1.5" />}
              {isStopping ? '停止中' : '停止'}
            </Button>
          ) : (
            <Button size="sm" onClick={() => onStart(profile.profileId)} title={isStarting ? '启动中' : '启动'} loading={isStarting}>
              {!isStarting && <Play className="w-4 h-4 fill-current mr-1.5" />}
              {isStarting ? '启动中' : '启动'}
            </Button>
          )}
          <span className="w-px h-4 bg-[var(--color-border-muted)] mx-1"></span>
          <Button size="sm" variant="ghost" onClick={() => onRestart(profile.profileId)} title="重启" className="px-3" disabled={isBusy}><RotateCcw className="w-4 h-4 mr-1.5" />重启</Button>
          <Button size="sm" variant="ghost" onClick={() => onOpenKeywords(profile)} title="关键字管理" className="px-3" disabled={isBusy}><Key className="w-4 h-4 mr-1.5" />关键字</Button>
          <Link to={`/browser/edit/${profile.profileId}`}><Button size="sm" variant="ghost" title="配置" className="px-3" disabled={isBusy}><Settings className="w-4 h-4 mr-1.5" />配置</Button></Link>
          <Button size="sm" variant="ghost" onClick={() => onOpenCopy(profile)} title="克隆" className="px-3" disabled={isBusy}><Copy className="w-4 h-4 mr-1.5" />克隆</Button>
          <Button size="sm" variant="ghost" onClick={() => onDelete(profile.profileId)} title="删除" className="px-3 text-red-500 hover:text-red-600 hover:bg-red-50" disabled={isBusy}><Trash2 className="w-4 h-4 mr-1.5" />删除</Button>
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 py-2 shrink-0">
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-[var(--color-text-muted)] font-medium">内核版本</span>
          <span className="text-xs text-[var(--color-text-primary)]">{coreLabel}</span>
        </div>
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-[var(--color-text-muted)] font-medium">代理配置</span>
          <span className="text-xs text-[var(--color-text-primary)] truncate" title={formatProxyLabel(profile, proxy)}>
            {formatProxyLabel(profile, proxy)}
          </span>
        </div>
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-[var(--color-text-muted)] font-medium">快捷配置码</span>
          <div className="mt-0.5"><LaunchCodeCell profileId={profile.profileId} code={profile.launchCode || ''} onRefresh={onRefreshProfiles} /></div>
        </div>
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-[var(--color-text-muted)] font-medium">时间</span>
          <span className="text-xs text-[var(--color-text-primary)]">创建 {formatTime(profile.createdAt)}</span>
          <span className="text-xs text-[var(--color-text-secondary)]">打开 {formatTime(profile.lastStartAt)}</span>
        </div>
      </div>

      <div className="border-t border-[var(--color-border-muted)]/50 pt-2 flex items-start gap-2 flex-1 min-h-0">
        <span className="text-xs font-medium text-[var(--color-text-primary)] shrink-0 pt-0.5">系统关键字</span>
        <div className="flex-1 min-h-0 overflow-y-auto pr-1">
          <KeywordInlineRow keywords={profile.keywords || []} />
        </div>
      </div>
    </div>
  )
}

export function BrowserProfilesPanel({
  loading,
  viewMode,
  profiles,
  totalCount,
  proxies,
  selectedIds,
  sortColumn,
  sortOrder,
  currentPage,
  pageSize,
  resolveProfileCore,
  getProfileCoreLabel,
  getProfileStatus,
  isProfileStarting,
  isProfileStopping,
  isProfileBusy,
  onToggleSelect,
  onSelectAll,
  onDeselectAll,
  onSortChange,
  onPageChange,
  onPageSizeChange,
  onRefreshProfiles,
  onStart,
  onStop,
  onRestart,
  onOpenKeywords,
  onOpenCopy,
  onDelete,
}: BrowserProfilesPanelProps) {
  const allSelected = totalCount > 0 && selectedIds.size === totalCount
  const partiallySelected = selectedIds.size > 0 && selectedIds.size < totalCount
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize))

  const columns: TableColumn<BrowserProfile>[] = [
    {
      key: 'selection',
      title: (
        <input
          type="checkbox"
          className="w-4 h-4 rounded cursor-pointer accent-[var(--color-accent)]"
          checked={allSelected}
          ref={(input) => {
            if (input) {
              input.indeterminate = partiallySelected
            }
          }}
          onChange={(event) => {
            if (event.target.checked) {
              onSelectAll()
            } else {
              onDeselectAll()
            }
          }}
        />
      ),
      width: 40,
      render: (_, record) => (
        <input
          type="checkbox"
          className="w-4 h-4 rounded cursor-pointer accent-[var(--color-accent)]"
          checked={selectedIds.has(record.profileId)}
          onChange={() => onToggleSelect(record.profileId)}
        />
      ),
    },
    {
      key: 'profileId',
      title: '序号',
      width: 132,
      sortable: true,
      render: (value) => (
        <span
          className="inline-flex max-w-[112px] items-center justify-center truncate rounded-md bg-[var(--color-bg-muted)] px-2.5 py-1 text-xs font-semibold text-[var(--color-text-primary)]"
          title={value}
        >
          {shortProfileId(value || '')}
        </span>
      ),
    },
    {
      key: 'profileName',
      title: '实例名称',
      width: 180,
      sortable: true,
      render: (value, record) => (
        <div className="flex flex-col gap-1">
          <Link className="text-[var(--color-accent)] text-sm font-medium hover:underline" to={`/browser/detail/${record.profileId}`}>
            {value}
          </Link>
          {record.tags && record.tags.length > 0 && (
            <div className="flex gap-1 flex-wrap">
              {record.tags.map(tag => <Badge variant="default" key={tag}>{tag}</Badge>)}
            </div>
          )}
        </div>
      ),
    },
    {
      key: 'running',
      title: '状态',
      width: 92,
      render: (_, record) => {
        const status = getProfileStatus(record)
        return <span className="whitespace-nowrap"><Badge variant={status.variant} dot>{status.label}</Badge></span>
      },
    },
    {
      key: 'coreId',
      title: '核心',
      width: 110,
      render: (_, record) => <span className="whitespace-nowrap text-xs">{getProfileCoreLabel(record)}</span>,
    },
    {
      key: 'proxyId',
      title: '代理',
      width: 120,
      render: (value, record) => {
        const proxy = proxies.find(item => item.proxyId === value)
        return <span className="whitespace-nowrap text-xs" title={formatProxyLabel(record, proxy)}>{formatProxyLabel(record, proxy)}</span>
      },
    },
    {
      key: 'launchCode',
      title: '快捷打开码',
      width: 150,
      render: (value, record) => <LaunchCodeCell profileId={record.profileId} code={value || ''} onRefresh={onRefreshProfiles} />,
    },
    {
      key: 'keywords',
      title: '关键字',
      width: 200,
      render: (value) => <KeywordInlineRow keywords={value || []} />,
    },
    {
      key: 'lastStartAt',
      title: '时间',
      width: 176,
      sortable: true,
      render: (_, record) => (
        <div className="space-y-0.5 text-xs leading-5">
          <div>
            <span className="text-[var(--color-text-muted)]">创建 </span>
            <span className="text-[var(--color-text-secondary)]">{formatTime(record.createdAt)}</span>
          </div>
          <div>
            <span className="text-[var(--color-text-muted)]">打开 </span>
            <span className="text-[var(--color-text-primary)]">{formatTime(record.lastStartAt)}</span>
          </div>
        </div>
      ),
    },
    {
      key: 'actions',
      title: '操作',
      align: 'right',
      width: 220,
      render: (_, record) => {
        const isStarting = isProfileStarting(record.profileId)
        const isStopping = isProfileStopping(record.profileId)
        const isBusy = isProfileBusy(record.profileId)

        return (
          <div className="flex justify-end gap-1">
            {record.running ? (
              <Button size="sm" variant="secondary" onClick={() => onStop(record.profileId)} title="停止" loading={isStopping}>
                {!isStopping && <Square className="w-3.5 h-3.5" />}
              </Button>
            ) : (
              <Button size="sm" onClick={() => onStart(record.profileId)} title="启动" loading={isStarting}>
                {!isStarting && <Play className="w-3.5 h-3.5 fill-current" />}
              </Button>
            )}
            <Button size="sm" variant="ghost" onClick={() => onRestart(record.profileId)} title="重启" disabled={isBusy}><RotateCcw className="w-3.5 h-3.5" /></Button>
            <Button size="sm" variant="ghost" onClick={() => onOpenKeywords(record)} title="关键字" disabled={isBusy}><Key className="w-3.5 h-3.5" /></Button>
            <Link to={`/browser/edit/${record.profileId}`}><Button size="sm" variant="ghost" title="配置" disabled={isBusy}><Settings className="w-3.5 h-3.5" /></Button></Link>
            <Button size="sm" variant="ghost" onClick={() => onOpenCopy(record)} title="克隆" disabled={isBusy}><Copy className="w-3.5 h-3.5" /></Button>
            <Button size="sm" variant="ghost" onClick={() => onDelete(record.profileId)} title="删除" disabled={isBusy}><Trash2 className="w-3.5 h-3.5 text-red-500" /></Button>
          </div>
        )
      },
    },
  ]

  return (
    <Card padding="none">
      <div className="overflow-auto" style={{ maxHeight: 'calc(100vh - 320px)' }}>
        {loading ? (
          <div className="py-16 flex items-center justify-center text-sm text-[var(--color-text-muted)]">加载中...</div>
        ) : profiles.length === 0 ? (
          <div className="py-16 flex items-center justify-center text-sm text-[var(--color-text-muted)]">暂无数据</div>
        ) : viewMode === 'table' ? (
          <Table
            columns={columns}
            data={profiles}
            rowKey="profileId"
            tableMinWidth="1120px"
            onSort={onSortChange}
            sortColumn={sortColumn}
            sortOrder={sortOrder}
          />
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 min-h-[500px] p-4 items-start content-start">
            {profiles.map((profile) => (
              <BrowserProfileCard
                key={profile.profileId}
                profile={profile}
                proxy={proxies.find(item => item.proxyId === profile.proxyId)}
                isSelected={selectedIds.has(profile.profileId)}
                status={getProfileStatus(profile)}
                coreLabel={resolveProfileCore(profile)?.coreName || getProfileCoreLabel(profile)}
                isStarting={isProfileStarting(profile.profileId)}
                isStopping={isProfileStopping(profile.profileId)}
                isBusy={isProfileBusy(profile.profileId)}
                onToggleSelect={onToggleSelect}
                onRefreshProfiles={onRefreshProfiles}
                onStart={onStart}
                onStop={onStop}
                onRestart={onRestart}
                onOpenKeywords={onOpenKeywords}
                onOpenCopy={onOpenCopy}
                onDelete={onDelete}
              />
            ))}
          </div>
        )}
      </div>
      {!loading && totalCount > 0 && (
        <PaginationBar
          totalCount={totalCount}
          currentPage={currentPage}
          pageSize={pageSize}
          totalPages={totalPages}
          onPageChange={onPageChange}
          onPageSizeChange={onPageSizeChange}
        />
      )}
    </Card>
  )
}

function PaginationBar({
  totalCount,
  currentPage,
  pageSize,
  totalPages,
  onPageChange,
  onPageSizeChange,
}: {
  totalCount: number
  currentPage: number
  pageSize: number
  totalPages: number
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
}) {
  const pages = buildVisiblePages(currentPage, totalPages)
  const canPrev = currentPage > 1
  const canNext = currentPage < totalPages

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] px-4 py-3 text-sm text-[var(--color-text-secondary)]">
      <div className="flex items-center gap-3">
        <span>共 {totalCount} 条</span>
        <select
          className="h-8 rounded-md border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent)]"
          value={pageSize}
          onChange={(event) => onPageSizeChange(Number(event.target.value))}
        >
          {[10, 20, 50, 100].map((size) => (
            <option key={size} value={size}>{size}条/页</option>
          ))}
        </select>
      </div>

      <div className="flex items-center gap-1">
        <button
          type="button"
          className="flex h-8 w-8 items-center justify-center rounded-md border border-transparent text-[var(--color-text-muted)] transition-colors hover:border-[var(--color-border-default)] hover:text-[var(--color-text-primary)] disabled:cursor-not-allowed disabled:opacity-40"
          disabled={!canPrev}
          onClick={() => canPrev && onPageChange(currentPage - 1)}
          title="上一页"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>

        {pages.map((page, index) => (
          page === 'ellipsis' ? (
            <span key={`ellipsis-${index}`} className="px-2 text-[var(--color-text-muted)]">...</span>
          ) : (
            <button
              key={page}
              type="button"
              className={`h-8 min-w-8 rounded-md px-2 text-sm font-medium transition-colors ${
                page === currentPage
                  ? 'bg-[var(--color-accent)] text-white'
                  : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-muted)] hover:text-[var(--color-text-primary)]'
              }`}
              onClick={() => onPageChange(page)}
            >
              {page}
            </button>
          )
        ))}

        <button
          type="button"
          className="flex h-8 w-8 items-center justify-center rounded-md border border-transparent text-[var(--color-text-muted)] transition-colors hover:border-[var(--color-border-default)] hover:text-[var(--color-text-primary)] disabled:cursor-not-allowed disabled:opacity-40"
          disabled={!canNext}
          onClick={() => canNext && onPageChange(currentPage + 1)}
          title="下一页"
        >
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  )
}

function buildVisiblePages(currentPage: number, totalPages: number): Array<number | 'ellipsis'> {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => index + 1)
  }

  const pages = new Set([1, totalPages, currentPage - 1, currentPage, currentPage + 1])
  const sorted = Array.from(pages)
    .filter((page) => page >= 1 && page <= totalPages)
    .sort((a, b) => a - b)

  return sorted.reduce<Array<number | 'ellipsis'>>((result, page) => {
    const previous = result[result.length - 1]
    if (typeof previous === 'number' && page - previous > 1) {
      result.push('ellipsis')
    }
    result.push(page)
    return result
  }, [])
}
