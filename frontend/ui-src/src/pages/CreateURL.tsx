import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { urls } from '../api/client'
import { Copy, Check } from 'lucide-react'
import TagInput from '../components/TagInput'

export default function CreateURL() {
  const navigate = useNavigate()
  const [form, setForm] = useState({
    longUrl: '',
    customSlug: '',
    title: '',
    ttl: '0',
    maxVisits: '',
    redirectType: '302',
    isCrawlable: true,
    domain: '',
    tags: [] as string[],
  })
  const [result, setResult] = useState<{ code: string } | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const data: Record<string, unknown> = {
        longUrl: form.longUrl,
        ttl: form.ttl,
        redirectType: Number(form.redirectType),
        isCrawlable: form.isCrawlable,
      }
      if (form.customSlug) data.customSlug = form.customSlug
      if (form.title) data.title = form.title
      if (form.maxVisits) data.maxVisits = Number(form.maxVisits)
      if (form.domain) data.domain = form.domain
      if (form.tags.length > 0) data.tags = form.tags

      const res = await urls.create(data)
      setResult({ code: res.code as string })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed')
    } finally {
      setLoading(false)
    }
  }

  const shortURL = result ? `${window.location.origin}/${result.code}` : ''

  const copyURL = () => {
    navigator.clipboard.writeText(shortURL)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (result) {
    return (
      <div className="max-w-lg mx-auto">
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6 space-y-4">
          <h2 className="text-lg font-semibold text-green-700 dark:text-green-400">URL Created!</h2>
          <div className="flex items-center gap-2 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <code className="flex-1 text-sm font-mono text-blue-600 break-all">{shortURL}</code>
            <button
              onClick={copyURL}
              className="p-2 rounded hover:bg-gray-200 transition-colors"
              title="Copy"
            >
              {copied ? <Check className="w-4 h-4 text-green-600" /> : <Copy className="w-4 h-4" />}
            </button>
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => navigate(`/urls/${result.code}`)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
            >
              View Analytics
            </button>
            <button
              onClick={() => {
                setResult(null)
                setForm({ ...form, longUrl: '', customSlug: '', title: '', tags: [] })
              }}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 text-sm"
            >
              Create Another
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-lg mx-auto">
      <form onSubmit={handleSubmit} className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6 space-y-4">
        {error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{error}</div>
        )}

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Long URL <span className="text-red-500">*</span>
          </label>
          <input
            type="url"
            value={form.longUrl}
            onChange={(e) => setForm({ ...form, longUrl: e.target.value })}
            required
            placeholder="https://example.com/very/long/url"
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Custom Slug</label>
            <input
              type="text"
              value={form.customSlug}
              onChange={(e) => setForm({ ...form, customSlug: e.target.value })}
              placeholder="my-link"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Title</label>
            <input
              type="text"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              placeholder="My Link"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">TTL</label>
            <select
              value={form.ttl}
              onChange={(e) => setForm({ ...form, ttl: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            >
              <option value="0">Never expires</option>
              <option value="300">5 minutes</option>
              <option value="3600">1 hour</option>
              <option value="86400">24 hours</option>
              <option value="172800">48 hours</option>
              <option value="604800">7 days</option>
              <option value="2592000">30 days</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Max Visits</label>
            <input
              type="number"
              value={form.maxVisits}
              onChange={(e) => setForm({ ...form, maxVisits: e.target.value })}
              placeholder="Unlimited"
              min="1"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Redirect Type</label>
            <select
              value={form.redirectType}
              onChange={(e) => setForm({ ...form, redirectType: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            >
              <option value="301">301 — Permanent</option>
              <option value="302">302 — Found (default)</option>
              <option value="307">307 — Temporary</option>
              <option value="308">308 — Permanent (strict)</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain</label>
            <input
              type="text"
              value={form.domain}
              onChange={(e) => setForm({ ...form, domain: e.target.value })}
              placeholder="Optional"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Tags</label>
          <TagInput
            value={form.tags}
            onChange={(tags) => setForm({ ...form, tags })}
          />
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="crawlable"
            checked={form.isCrawlable}
            onChange={(e) => setForm({ ...form, isCrawlable: e.target.checked })}
            className="rounded border-gray-300"
          />
          <label htmlFor="crawlable" className="text-sm text-gray-700 dark:text-gray-300">
            Allow search engine crawling
          </label>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full py-2.5 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          {loading ? 'Creating...' : 'Shorten URL'}
        </button>
      </form>
    </div>
  )
}
