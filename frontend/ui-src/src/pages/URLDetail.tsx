import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import QRCodeStyling from 'qr-code-styling'
import { urls, analytics } from '../api/client'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from 'recharts'
import { ArrowLeft, ExternalLink, Copy, QrCode } from 'lucide-react'

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16']

export default function URLDetail() {
  const { code } = useParams<{ code: string }>()
  const [urlData, setUrlData] = useState<Record<string, unknown> | null>(null)
  const [summary, setSummary] = useState<Record<string, unknown> | null>(null)
  const [dateData, setDateData] = useState<{ date: string; visits: number }[]>([])
  const [browserData, setBrowserData] = useState<{ value: string; visits: number }[]>([])
  const [countryData, setCountryData] = useState<{ value: string; visits: number }[]>([])
  const [deviceData, setDeviceData] = useState<{ value: string; visits: number }[]>([])
  const [refererData, setRefererData] = useState<{ value: string; visits: number }[]>([])
  const [excludeBots, setExcludeBots] = useState(true)
  const [showQR, setShowQR] = useState(false)
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    if (!code) return
    setLoading(true)
    try {
      const [urlRes, summaryRes, dateRes, browserRes, countryRes, deviceRes, refererRes] =
        await Promise.all([
          urls.stats(code),
          analytics.summary(code, excludeBots),
          analytics.byDate(code, excludeBots),
          analytics.byField(code, 'browser', excludeBots),
          analytics.byField(code, 'country', excludeBots),
          analytics.byField(code, 'device_type', excludeBots),
          analytics.byField(code, 'referer', excludeBots, 5),
        ])
      setUrlData(urlRes)
      setSummary(summaryRes)
      setDateData(
        (dateRes.entries || []).map((e) => ({ date: e.date, visits: Number(e.visits) }))
      )
      setBrowserData(
        (browserRes.entries || []).map((e) => ({ value: e.value, visits: Number(e.visits) }))
      )
      setCountryData(
        (countryRes.entries || []).map((e) => ({ value: e.value, visits: Number(e.visits) }))
      )
      setDeviceData(
        (deviceRes.entries || []).map((e) => ({ value: e.value, visits: Number(e.visits) }))
      )
      setRefererData(
        (refererRes.entries || []).map((e) => ({ value: e.value, visits: Number(e.visits) }))
      )
    } catch (err) {
      console.error('Failed to load analytics', err)
    } finally {
      setLoading(false)
    }
  }, [code, excludeBots])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  if (loading) {
    return <div className="text-center py-12 text-gray-500">Loading analytics...</div>
  }

  if (!urlData) {
    return <div className="text-center py-12 text-gray-500">URL not found</div>
  }

  const shortURL = `${window.location.origin}/${code}`

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link to="/" className="p-1.5 rounded hover:bg-gray-200">
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1 min-w-0">
          <h2 className="text-xl font-bold text-gray-900 font-mono">/{code}</h2>
          <a
            href={urlData.longUrl as string}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-gray-500 hover:text-gray-700 truncate block"
          >
            {(urlData.title as string) || (urlData.longUrl as string)}
            <ExternalLink className="w-3 h-3 inline ml-1" />
          </a>
        </div>
        <button
          onClick={() => navigator.clipboard.writeText(shortURL)}
          className="p-2 rounded border border-gray-300 hover:bg-gray-50"
          title="Copy short URL"
        >
          <Copy className="w-4 h-4" />
        </button>
        <button
          onClick={() => setShowQR(!showQR)}
          className="p-2 rounded border border-gray-300 hover:bg-gray-50"
          title="QR Code"
        >
          <QrCode className="w-4 h-4" />
        </button>
      </div>

      {showQR && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
          <QRPanel url={shortURL} />
        </div>
      )}

      {/* Summary cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {[
          { label: 'Total Visits', value: Number(summary?.totalVisits || 0) },
          { label: 'Unique Visitors', value: Number(summary?.uniqueVisitors || 0) },
          { label: 'Human Visits', value: Number(summary?.humanVisits || 0) },
          { label: 'Bot Visits', value: Number(summary?.botVisits || 0) },
        ].map((card) => (
          <div key={card.label} className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
            <p className="text-sm text-gray-500">{card.label}</p>
            <p className="text-2xl font-bold text-gray-900">{card.value.toLocaleString()}</p>
          </div>
        ))}
      </div>

      {/* Bot filter toggle */}
      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          id="excludeBots"
          checked={excludeBots}
          onChange={(e) => setExcludeBots(e.target.checked)}
          className="rounded border-gray-300"
        />
        <label htmlFor="excludeBots" className="text-sm text-gray-700">
          Exclude bot traffic
        </label>
      </div>

      {/* Visits over time */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
        <h3 className="text-sm font-medium text-gray-700 mb-4">Visits Over Time</h3>
        {dateData.length > 0 ? (
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={dateData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <Tooltip />
              <Bar dataKey="visits" fill="#3b82f6" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <p className="text-center py-8 text-gray-400">No visit data yet</p>
        )}
      </div>

      {/* Charts grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Browsers */}
        <ChartCard title="Browsers" data={browserData} />
        {/* Countries */}
        <ChartCard title="Countries" data={countryData} />
        {/* Devices */}
        <ChartCard title="Devices" data={deviceData} />
        {/* Referrers */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-4">Top Referrers</h3>
          {refererData.length > 0 ? (
            <div className="space-y-2">
              {refererData.map((item, i) => (
                <div key={i} className="flex items-center justify-between text-sm">
                  <span className="text-gray-600 truncate flex-1 mr-2">{item.value || 'Direct'}</span>
                  <span className="font-medium text-gray-900">{item.visits.toLocaleString()}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-center py-8 text-gray-400">No referrer data</p>
          )}
        </div>
      </div>

      {/* URL metadata */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
        <h3 className="text-sm font-medium text-gray-700 mb-3">URL Details</h3>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <dt className="text-gray-500">Status</dt>
          <dd>{(urlData.isActive as boolean) ? '✓ Active' : '✗ Inactive'}</dd>
          <dt className="text-gray-500">Redirect Type</dt>
          <dd>{String(urlData.redirectType || 302)}</dd>
          <dt className="text-gray-500">Max Visits</dt>
          <dd>{Number(urlData.maxVisits || 0) > 0 ? String(urlData.maxVisits) : 'Unlimited'}</dd>
          <dt className="text-gray-500">Created</dt>
          <dd>
            {urlData.createdAt ? new Date(urlData.createdAt as string).toLocaleString() : '—'}
          </dd>
          <dt className="text-gray-500">Expires</dt>
          <dd>
            {urlData.expiresAt ? new Date(urlData.expiresAt as string).toLocaleString() : 'Never'}
          </dd>
          <dt className="text-gray-500">Tags</dt>
          <dd>
            {((urlData.tags as string[]) || []).join(', ') || 'None'}
          </dd>
        </dl>
      </div>
    </div>
  )
}

// --- QR Panel ---

type DotStyle = 'square' | 'rounded' | 'dots' | 'classy' | 'classy-rounded' | 'extra-rounded'

const DOT_STYLES: { label: string; value: DotStyle }[] = [
  { label: 'Square', value: 'square' },
  { label: 'Rounded', value: 'rounded' },
  { label: 'Dots', value: 'dots' },
  { label: 'Classy', value: 'classy' },
  { label: 'Classy Rounded', value: 'classy-rounded' },
  { label: 'Extra Rounded', value: 'extra-rounded' },
]

function QRPanel({ url }: { url: string }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const qrRef = useRef<QRCodeStyling | null>(null)
  const [dotStyle, setDotStyle] = useState<DotStyle>('rounded')
  const [fgColor, setFgColor] = useState('#000000')
  const [bgColor, setBgColor] = useState('#ffffff')

  useEffect(() => {
    if (!containerRef.current) return
    const qr = new QRCodeStyling({
      width: 220,
      height: 220,
      data: url,
      dotsOptions: { color: fgColor, type: dotStyle },
      backgroundOptions: { color: bgColor },
      cornersSquareOptions: { type: 'extra-rounded' },
      cornersDotOptions: { type: 'dot' },
      qrOptions: { errorCorrectionLevel: 'M' },
    })
    containerRef.current.innerHTML = ''
    qr.append(containerRef.current)
    qrRef.current = qr
  }, [url, dotStyle, fgColor, bgColor])

  return (
    <div className="flex flex-col items-center gap-4">
      <div ref={containerRef} />
      <div className="flex flex-wrap items-center justify-center gap-3 text-sm">
        <select
          value={dotStyle}
          onChange={(e) => setDotStyle(e.target.value as DotStyle)}
          className="border border-gray-300 rounded px-2 py-1 text-sm"
        >
          {DOT_STYLES.map((s) => (
            <option key={s.value} value={s.value}>{s.label}</option>
          ))}
        </select>
        <label className="flex items-center gap-1.5 text-gray-600">
          <input
            type="color"
            value={fgColor}
            onChange={(e) => setFgColor(e.target.value)}
            className="w-6 h-6 rounded cursor-pointer border border-gray-300"
          />
          Color
        </label>
        <label className="flex items-center gap-1.5 text-gray-600">
          <input
            type="color"
            value={bgColor}
            onChange={(e) => setBgColor(e.target.value)}
            className="w-6 h-6 rounded cursor-pointer border border-gray-300"
          />
          Background
        </label>
        <button
          onClick={() => qrRef.current?.download({ name: `qr-${url.split('/').pop()}`, extension: 'png' })}
          className="px-3 py-1 rounded border border-gray-300 hover:bg-gray-50"
        >
          Download PNG
        </button>
      </div>
    </div>
  )
}

// --- Charts ---

function ChartCard({ title, data }: { title: string; data: { value: string; visits: number }[] }) {
  if (data.length === 0) {
    return (
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
        <h3 className="text-sm font-medium text-gray-700 mb-4">{title}</h3>
        <p className="text-center py-8 text-gray-400">No data</p>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
      <h3 className="text-sm font-medium text-gray-700 mb-4">{title}</h3>
      <div className="flex items-center gap-4">
        <ResponsiveContainer width="50%" height={160}>
          <PieChart>
            <Pie
              data={data}
              dataKey="visits"
              nameKey="value"
              cx="50%"
              cy="50%"
              outerRadius={65}
              innerRadius={35}
            >
              {data.map((_, i) => (
                <Cell key={i} fill={COLORS[i % COLORS.length]} />
              ))}
            </Pie>
            <Tooltip />
          </PieChart>
        </ResponsiveContainer>
        <div className="flex-1 space-y-1.5">
          {data.slice(0, 5).map((item, i) => (
            <div key={i} className="flex items-center gap-2 text-xs">
              <div
                className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                style={{ backgroundColor: COLORS[i % COLORS.length] }}
              />
              <span className="text-gray-600 truncate flex-1">{item.value}</span>
              <span className="font-medium text-gray-900">{item.visits}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
