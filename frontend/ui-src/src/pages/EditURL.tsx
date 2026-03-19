import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { urls, admin } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import TagInput from '../components/TagInput'

interface URLForm {
  longUrl: string
  title: string
  ttl: string
  maxVisits: string
  redirectType: string
  isCrawlable: boolean
  isActive: boolean
  tags: string[]
}

interface UserOption {
  id: number
  email: string
  name: string
}

export default function EditURL() {
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const [form, setForm] = useState<URLForm>({
    longUrl: '',
    title: '',
    ttl: '0',
    maxVisits: '',
    redirectType: '302',
    isCrawlable: true,
    isActive: true,
    tags: [] as string[],
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  // Owner assignment (admin only)
  const [currentOwnerEmail, setCurrentOwnerEmail] = useState<string>('')
  const [currentOwnerId, setCurrentOwnerId] = useState<number | null>(null)
  const [selectedOwnerId, setSelectedOwnerId] = useState<number | null>(null)
  const [userOptions, setUserOptions] = useState<UserOption[]>([])
  const [userSearch, setUserSearch] = useState('')

  useEffect(() => {
    if (!code) return

    const loadData = async () => {
      try {
        if (isAdmin) {
          // Admin uses the admin endpoint to also get createdByEmail/createdByUserId
          const [adminRec, usersRes] = await Promise.all([
            admin.urls.get(code),
            admin.users.list({ page: 1, page_size: 200 }),
          ])
          const r = adminRec as Record<string, unknown>
          setForm({
            longUrl: (r.longUrl as string) || '',
            title: (r.title as string) || '',
            ttl: '0',
            maxVisits: r.maxVisits ? String(r.maxVisits) : '',
            redirectType: String(r.redirectType || 302),
            isCrawlable: (r.isCrawlable as boolean) ?? true,
            isActive: (r.isActive as boolean) ?? true,
            tags: (r.tags as string[]) || [],
          })
          const ownerEmail = (r.createdByEmail as string) || ''
          const ownerIdRaw = r.createdByUserId
          const ownerId = ownerIdRaw != null ? Number(ownerIdRaw) : null
          setCurrentOwnerEmail(ownerEmail)
          setCurrentOwnerId(ownerId)
          setSelectedOwnerId(ownerId)

          const opts: UserOption[] = ((usersRes.users || []) as Array<Record<string, unknown>>).map((u) => ({
            id: Number(u.id),
            email: String(u.email),
            name: String(u.name || ''),
          }))
          setUserOptions(opts)
        } else {
          const res = await urls.get(code)
          const r = res as Record<string, unknown>
          setForm({
            longUrl: (r.longUrl as string) || '',
            title: (r.title as string) || '',
            ttl: '0',
            maxVisits: r.maxVisits ? String(r.maxVisits) : '',
            redirectType: String(r.redirectType || 302),
            isCrawlable: (r.isCrawlable as boolean) ?? true,
            isActive: (r.isActive as boolean) ?? true,
            tags: (r.tags as string[]) || [],
          })
        }
      } catch {
        setError('Failed to load URL')
      } finally {
        setLoading(false)
      }
    }

    loadData()
  }, [code, isAdmin])

  const filteredUsers = userSearch.trim()
    ? userOptions.filter(
        (u) =>
          u.email.toLowerCase().includes(userSearch.toLowerCase()) ||
          u.name.toLowerCase().includes(userSearch.toLowerCase())
      )
    : userOptions

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!code) return
    setError('')
    setSaving(true)
    try {
      const data: Record<string, unknown> = {
        longUrl: form.longUrl,
        redirectType: Number(form.redirectType),
        isCrawlable: { value: form.isCrawlable },
        isActive: { value: form.isActive },
      }
      if (form.title) data.title = form.title
      if (form.maxVisits) data.maxVisits = Number(form.maxVisits)
      if (form.ttl && form.ttl !== '0') data.ttl = Number(form.ttl)
      data.tags = form.tags

      await urls.update(code, data)

      // Admin: reassign owner if changed
      if (isAdmin && selectedOwnerId !== currentOwnerId) {
        await admin.urls.assign(code, selectedOwnerId ?? 0)
      }

      navigate(`/urls/${code}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400">
        Loading...
      </div>
    )
  }

  return (
    <div className="max-w-lg mx-auto">
      <form
        onSubmit={handleSubmit}
        className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6 space-y-4"
      >
        <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">
          Edit <span className="font-mono text-blue-600">/{code}</span>
        </h2>

        {error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
            {error}
          </div>
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
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Title
          </label>
          <input
            type="text"
            value={form.title}
            onChange={(e) => setForm({ ...form, title: e.target.value })}
            placeholder="Optional"
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Extend TTL
            </label>
            <select
              value={form.ttl}
              onChange={(e) => setForm({ ...form, ttl: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none"
            >
              <option value="0">No change</option>
              <option value="-1">Never expire</option>
              <option value="86400">+24 hours</option>
              <option value="604800">+7 days</option>
              <option value="2592000">+30 days</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Max Visits
            </label>
            <input
              type="number"
              value={form.maxVisits}
              onChange={(e) => setForm({ ...form, maxVisits: e.target.value })}
              placeholder="Unlimited"
              min="1"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Redirect Type
          </label>
          <select
            value={form.redirectType}
            onChange={(e) => setForm({ ...form, redirectType: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none"
          >
            <option value="301">301 — Permanent</option>
            <option value="302">302 — Found (default)</option>
            <option value="307">307 — Temporary</option>
            <option value="308">308 — Permanent (strict)</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Tags
          </label>
          <TagInput
            value={form.tags}
            onChange={(tags) => setForm({ ...form, tags })}
          />
        </div>

        <div className="flex flex-col gap-2">
          <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 cursor-pointer">
            <input
              type="checkbox"
              checked={form.isCrawlable}
              onChange={(e) => setForm({ ...form, isCrawlable: e.target.checked })}
              className="rounded border-gray-300"
            />
            Allow search engine crawling
          </label>
          <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 cursor-pointer">
            <input
              type="checkbox"
              checked={form.isActive}
              onChange={(e) => setForm({ ...form, isActive: e.target.checked })}
              className="rounded border-gray-300"
            />
            Active (redirects enabled)
          </label>
        </div>

        {isAdmin && (
          <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Owner
              {currentOwnerEmail && (
                <span className="ml-2 font-normal text-gray-400 dark:text-gray-500">
                  (currently {currentOwnerEmail})
                </span>
              )}
            </label>
            <input
              type="text"
              placeholder="Search by email or name..."
              value={userSearch}
              onChange={(e) => setUserSearch(e.target.value)}
              className="w-full px-3 py-2 mb-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none text-sm"
            />
            <select
              value={selectedOwnerId ?? ''}
              onChange={(e) =>
                setSelectedOwnerId(e.target.value === '' ? null : Number(e.target.value))
              }
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
              size={Math.min(5, filteredUsers.length + 1)}
            >
              <option value="">— No owner (anonymous) —</option>
              {filteredUsers.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.email}{u.name ? ` (${u.name})` : ''}
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={saving}
            className="flex-1 py-2.5 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving...' : 'Save Changes'}
          </button>
          <button
            type="button"
            onClick={() => navigate(`/urls/${code}`)}
            className="px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}
