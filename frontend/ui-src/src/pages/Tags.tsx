import { useState, useEffect, useCallback } from 'react'
import { tags } from '../api/client'
import { Plus, Pencil, Trash2, BarChart3, X, Check } from 'lucide-react'

interface Tag {
  id: string
  name: string
  urlCount: string
}

interface TagStatsData {
  tag: Tag
  totalClicks: string
  uniqueUrls: string
}

export default function TagsPage() {
  const [tagList, setTagList] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)
  const [newTag, setNewTag] = useState('')
  const [editing, setEditing] = useState<{ name: string; newName: string } | null>(null)
  const [statsFor, setStatsFor] = useState<TagStatsData | null>(null)
  const [error, setError] = useState('')

  const fetchTags = useCallback(async () => {
    try {
      const res = await tags.list()
      setTagList(res.tags || [])
    } catch (err) {
      console.error('Failed to fetch tags', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchTags()
  }, [fetchTags])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newTag.trim()) return
    setError('')
    try {
      await tags.create(newTag.trim())
      setNewTag('')
      fetchTags()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed')
    }
  }

  const handleRename = async () => {
    if (!editing || !editing.newName.trim()) return
    setError('')
    try {
      await tags.rename(editing.name, editing.newName.trim())
      setEditing(null)
      fetchTags()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Rename failed')
    }
  }

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete tag "${name}"? This will remove it from all URLs.`)) return
    try {
      await tags.delete(name)
      fetchTags()
    } catch (err) {
      console.error('Delete failed', err)
    }
  }

  const handleStats = async (name: string) => {
    try {
      const res = await tags.stats(name)
      setStatsFor(res)
    } catch (err) {
      console.error('Stats failed', err)
    }
  }

  return (
    <div className="space-y-4 max-w-2xl">
      {/* Create tag */}
      <form onSubmit={handleCreate} className="flex gap-2">
        <input
          type="text"
          value={newTag}
          onChange={(e) => setNewTag(e.target.value)}
          placeholder="New tag name..."
          className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
        />
        <button
          type="submit"
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium flex items-center gap-1"
        >
          <Plus className="w-4 h-4" /> Add
        </button>
      </form>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{error}</div>
      )}

      {/* Stats modal */}
      {statsFor && (
        <div className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded-xl p-4 relative">
          <button
            onClick={() => setStatsFor(null)}
            className="absolute top-2 right-2 text-gray-400 hover:text-gray-600"
          >
            <X className="w-4 h-4" />
          </button>
          <h3 className="font-medium text-gray-900 dark:text-gray-100 mb-2">
            Stats for &ldquo;{statsFor.tag.name}&rdquo;
          </h3>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{Number(statsFor.uniqueUrls)}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400">URLs</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {Number(statsFor.totalClicks).toLocaleString()}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400">Total Clicks</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {Number(statsFor.uniqueUrls) > 0
                  ? Math.round(Number(statsFor.totalClicks) / Number(statsFor.uniqueUrls))
                  : 0}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400">Avg Clicks/URL</p>
            </div>
          </div>
        </div>
      )}

      {/* Tag list */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400">Loading...</div>
        ) : tagList.length === 0 ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400">No tags yet. Create one above.</div>
        ) : (
          <ul className="divide-y divide-gray-100 dark:divide-gray-700">
            {tagList.map((tag) => (
              <li key={tag.id} className="flex items-center px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800">
                {editing?.name === tag.name ? (
                  <div className="flex-1 flex items-center gap-2">
                    <input
                      type="text"
                      value={editing.newName}
                      onChange={(e) => setEditing({ ...editing, newName: e.target.value })}
                      onKeyDown={(e) => e.key === 'Enter' && handleRename()}
                      className="flex-1 px-2 py-1 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                      autoFocus
                    />
                    <button onClick={handleRename} className="text-green-600 hover:text-green-700">
                      <Check className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => setEditing(null)}
                      className="text-gray-400 hover:text-gray-600"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ) : (
                  <>
                    <span className="flex-1 text-sm font-medium text-gray-900 dark:text-gray-100">{tag.name}</span>
                    <span className="text-xs text-gray-500 dark:text-gray-400 mr-4">
                      {Number(tag.urlCount)} URL{Number(tag.urlCount) !== 1 ? 's' : ''}
                    </span>
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => handleStats(tag.name)}
                        className="p-1.5 text-gray-400 hover:text-blue-600 rounded"
                        title="View stats"
                      >
                        <BarChart3 className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => setEditing({ name: tag.name, newName: tag.name })}
                        className="p-1.5 text-gray-400 hover:text-yellow-600 rounded"
                        title="Rename"
                      >
                        <Pencil className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleDelete(tag.name)}
                        className="p-1.5 text-gray-400 hover:text-red-600 rounded"
                        title="Delete"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
