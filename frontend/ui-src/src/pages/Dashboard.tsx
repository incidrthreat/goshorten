import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { urls, admin } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import {
  Search,
  ExternalLink,
  BarChart3,
  Copy,
  ChevronLeft,
  ChevronRight,
  Trash2,
  Pencil,
  Power,
} from 'lucide-react'

interface URLItem {
  code: string
  longUrl: string
  title: string
  createdAt: string
  isActive: boolean
  totalClicks: string
  tags: string[]
  domain: string
  createdByEmail?: string
}

export default function Dashboard() {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const [items, setItems] = useState<URLItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [orderBy, setOrderBy] = useState('created_at')
  const [orderDir, setOrderDir] = useState('desc')
  const [loading, setLoading] = useState(true)
  const pageSize = 20

  const fetchURLs = useCallback(async () => {
    setLoading(true)
    try {
      const params = { page, pageSize, search, orderBy, orderDir }
      const res = isAdmin
        ? await admin.urls.list(params)
        : await urls.list(params)
      setItems((res.urls || []) as unknown as URLItem[])
      setTotal(res.total || 0)
    } catch (err) {
      console.error('Failed to fetch URLs', err)
    } finally {
      setLoading(false)
    }
  }, [page, search, orderBy, orderDir, isAdmin])

  useEffect(() => {
    fetchURLs()
  }, [fetchURLs])

  const handleDelete = async (code: string) => {
    if (!confirm(`Permanently delete "${code}"? This cannot be undone.`)) return
    try {
      await urls.delete(code)
      fetchURLs()
    } catch (err) {
      console.error('Delete failed', err)
    }
  }

  const handleToggleActive = async (item: URLItem) => {
    try {
      await urls.update(item.code, { isActive: { value: !item.isActive } })
      fetchURLs()
    } catch (err) {
      console.error('Toggle failed', err)
    }
  }

  const copyToClipboard = (code: string) => {
    navigator.clipboard.writeText(`${window.location.origin}/${code}`)
  }

  const totalPages = Math.ceil(total / pageSize)

  const toggleSort = (field: string) => {
    if (orderBy === field) {
      setOrderDir(orderDir === 'asc' ? 'desc' : 'asc')
    } else {
      setOrderBy(field)
      setOrderDir('desc')
    }
    setPage(1)
  }

  // Admin sees N+1 columns (adds Owner before Tags)
  const colSpan = isAdmin ? 7 : 6

  return (
    <div className="space-y-4">
      {/* Search + controls */}
      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search URLs or codes..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(1)
            }}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-black-900 dark:text-black-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
          />
        </div>
        <Link
          to="/create"
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium text-center"
        >
          + New URL
        </Link>
      </div>

      {/* Table */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
                <th
                  className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:text-gray-700 dark:hover:text-gray-200"
                  onClick={() => toggleSort('code')}
                >
                  Code {orderBy === 'code' && (orderDir === 'asc' ? '↑' : '↓')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Destination
                </th>
                <th
                  className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:text-gray-700 dark:hover:text-gray-200"
                  onClick={() => toggleSort('clicks')}
                >
                  Clicks {orderBy === 'clicks' && (orderDir === 'asc' ? '↑' : '↓')}
                </th>
                <th
                  className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:text-gray-700 dark:hover:text-gray-200"
                  onClick={() => toggleSort('created_at')}
                >
                  Created {orderBy === 'created_at' && (orderDir === 'asc' ? '↑' : '↓')}
                </th>
                {isAdmin && (
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Owner
                  </th>
                )}
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Tags
                </th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {loading ? (
                <tr>
                  <td colSpan={colSpan} className="px-4 py-8 text-center text-gray-500">
                    Loading...
                  </td>
                </tr>
              ) : items.length === 0 ? (
                <tr>
                  <td colSpan={colSpan} className="px-4 py-8 text-center text-gray-500">
                    No short URLs yet.{' '}
                    <Link to="/create" className="text-blue-600 hover:underline">
                      Create one
                    </Link>
                  </td>
                </tr>
              ) : (
                items.map((item) => (
                  <tr
                    key={item.code}
                    className={
                      item.isActive
                        ? 'hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors'
                        : 'bg-amber-50 dark:bg-amber-900/10 hover:bg-amber-100 dark:hover:bg-amber-900/20 opacity-75 transition-colors'
                    }
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Link
                          to={`/urls/${item.code}`}
                          className="font-mono text-sm font-medium text-blue-600 hover:underline"
                        >
                          /{item.code}
                        </Link>
                        <button
                          onClick={() => copyToClipboard(item.code)}
                          className="text-gray-400 hover:text-gray-600"
                          title="Copy short URL"
                        >
                          <Copy className="w-3.5 h-3.5" />
                        </button>
                        {!item.isActive && (
                          <span className="px-1.5 py-0.5 text-xs bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400 rounded font-medium">
                            inactive
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 max-w-xs">
                      <a
                        href={item.longUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 truncate block"
                        title={item.longUrl}
                      >
                        {item.title || item.longUrl}
                        <ExternalLink className="w-3 h-3 inline ml-1 opacity-50" />
                      </a>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400 font-medium">
                      {Number(item.totalClicks || 0).toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-500">
                      {item.createdAt
                        ? new Date(item.createdAt).toLocaleDateString()
                        : '—'}
                    </td>
                    {isAdmin && (
                      <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400 max-w-[160px] truncate">
                        {item.createdByEmail || '—'}
                      </td>
                    )}
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {(item.tags || []).map((tag) => (
                          <span
                            key={tag}
                            className="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded-full"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Link
                          to={`/urls/${item.code}`}
                          className="text-gray-400 hover:text-blue-600"
                          title="Analytics"
                        >
                          <BarChart3 className="w-4 h-4" />
                        </Link>
                        <Link
                          to={`/urls/${item.code}/edit`}
                          className="text-gray-400 hover:text-yellow-600"
                          title="Edit"
                        >
                          <Pencil className="w-4 h-4" />
                        </Link>
                        <button
                          onClick={() => handleToggleActive(item)}
                          className={item.isActive ? 'text-green-500 hover:text-amber-500' : 'text-gray-400 hover:text-green-500'}
                          title={item.isActive ? 'Deactivate' : 'Activate'}
                        >
                          <Power className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(item.code)}
                          className="text-gray-400 hover:text-red-600"
                          title="Delete permanently"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
            <span className="text-sm text-gray-500">
              {total} total &middot; page {page} of {totalPages}
            </span>
            <div className="flex gap-2">
              <button
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
                className="p-1.5 rounded border border-gray-300 dark:border-gray-600 hover:bg-white dark:hover:bg-gray-700 disabled:opacity-30"
              >
                <ChevronLeft className="w-4 h-4" />
              </button>
              <button
                onClick={() => setPage(Math.min(totalPages, page + 1))}
                disabled={page === totalPages}
                className="p-1.5 rounded border border-gray-300 dark:border-gray-600 hover:bg-white dark:hover:bg-gray-700 disabled:opacity-30"
              >
                <ChevronRight className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
