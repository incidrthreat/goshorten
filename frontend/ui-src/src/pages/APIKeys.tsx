import { useState, useEffect, useCallback } from 'react'
import { apiKeys } from '../api/client'
import { Plus, Trash2, Copy, Check, Eye, EyeOff, RefreshCw, Info } from 'lucide-react'

interface KeyItem {
  id: string
  label: string
  scopes: string
  createdAt: string
  revoked: boolean
  keyPrefix: string
}

const AVAILABLE_SCOPES = [
  {
    value: 'urls:read',
    label: 'URLs: Read',
    description: 'List, search, and resolve short URLs. View URL metadata, stats, visit analytics, and QR codes.',
  },
  {
    value: 'urls:write',
    label: 'URLs: Write',
    description: 'Create, update, and delete short URLs. Manage tags on URLs.',
  },
  {
    value: 'keys:manage',
    label: 'Keys: Manage',
    description: 'Create, list, roll, and revoke API keys. Full control over API key lifecycle.',
  },
]

export default function APIKeysPage() {
  const [keys, setKeys] = useState<KeyItem[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [label, setLabel] = useState('')
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set(['urls:read', 'urls:write']))
  const [newKey, setNewKey] = useState('')
  const [newKeyLabel, setNewKeyLabel] = useState('')
  const [copied, setCopied] = useState(false)
  const [showKey, setShowKey] = useState(false)
  const [error, setError] = useState('')
  const [rolling, setRolling] = useState<string | null>(null)

  const fetchKeys = useCallback(async () => {
    try {
      const res = await apiKeys.list()
      setKeys((res.keys || []) as unknown as KeyItem[])
    } catch (err) {
      console.error('Failed to fetch keys', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchKeys()
  }, [fetchKeys])

  const toggleScope = (scope: string) => {
    setSelectedScopes((prev) => {
      const next = new Set(prev)
      if (next.has(scope)) {
        next.delete(scope)
      } else {
        next.add(scope)
      }
      return next
    })
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (selectedScopes.size === 0) {
      setError('Select at least one scope')
      return
    }
    try {
      const scopes = Array.from(selectedScopes).join(',')
      const res = await apiKeys.create(label, scopes)
      setNewKey(res.plaintextKey)
      setNewKeyLabel(label)
      setShowCreate(false)
      setLabel('')
      setSelectedScopes(new Set(['urls:read', 'urls:write']))
      fetchKeys()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed')
    }
  }

  const handleRevoke = async (keyId: number) => {
    if (!confirm('Revoke this API key? This cannot be undone.')) return
    try {
      await apiKeys.revoke(keyId)
      fetchKeys()
    } catch (err) {
      console.error('Revoke failed', err)
    }
  }

  const handleRoll = async (keyId: number, keyLabel: string) => {
    if (!confirm(`Roll API key "${keyLabel}"? The old key will be revoked and a new one generated with the same label and scopes.`)) return
    setRolling(String(keyId))
    try {
      const res = await apiKeys.roll(keyId)
      setNewKey(res.plaintextKey)
      setNewKeyLabel(keyLabel)
      setShowKey(false)
      setCopied(false)
      fetchKeys()
    } catch (err) {
      console.error('Roll failed', err)
    } finally {
      setRolling(null)
    }
  }

  const copyKey = () => {
    navigator.clipboard.writeText(newKey)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-4 max-w-3xl">
      {/* New key display */}
      {newKey && (
        <div className="bg-green-50 border border-green-200 rounded-xl p-4 space-y-2">
          <p className="text-sm font-medium text-green-800">
            API key {newKeyLabel ? `"${newKeyLabel}" ` : ''}created. Copy it now — it won&apos;t be shown again.
          </p>
          <div className="flex items-center gap-2 p-2 bg-white rounded border border-green-300">
            <code className="flex-1 text-sm font-mono break-all">
              {showKey ? newKey : '\u2022'.repeat(40)}
            </code>
            <button onClick={() => setShowKey(!showKey)} className="text-gray-400 hover:text-gray-600">
              {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
            <button onClick={copyKey} className="text-gray-400 hover:text-gray-600">
              {copied ? <Check className="w-4 h-4 text-green-600" /> : <Copy className="w-4 h-4" />}
            </button>
          </div>
          <button
            onClick={() => { setNewKey(''); setNewKeyLabel('') }}
            className="text-sm text-green-700 hover:underline"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Create form */}
      {showCreate ? (
        <form
          onSubmit={handleCreate}
          className="bg-white rounded-xl shadow-sm border border-gray-200 p-5 space-y-4"
        >
          <h3 className="text-lg font-semibold text-gray-800">Create API Key</h3>
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {error}
            </div>
          )}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Label</label>
            <input
              type="text"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              required
              placeholder="e.g., CI/CD Pipeline, Mobile App"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Scopes</label>
            <div className="space-y-2">
              {AVAILABLE_SCOPES.map((scope) => (
                <label
                  key={scope.value}
                  className={`flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition ${
                    selectedScopes.has(scope.value)
                      ? 'border-blue-300 bg-blue-50'
                      : 'border-gray-200 bg-white hover:bg-gray-50'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={selectedScopes.has(scope.value)}
                    onChange={() => toggleScope(scope.value)}
                    className="mt-0.5 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  />
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-gray-800">{scope.label}</span>
                      <code className="text-xs px-1.5 py-0.5 bg-gray-100 text-gray-500 rounded">{scope.value}</code>
                    </div>
                    <p className="text-xs text-gray-500 mt-0.5">{scope.description}</p>
                  </div>
                </label>
              ))}
            </div>
          </div>
          <div className="flex gap-2 pt-1">
            <button
              type="submit"
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-medium"
            >
              Create Key
            </button>
            <button
              type="button"
              onClick={() => { setShowCreate(false); setError('') }}
              className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm"
            >
              Cancel
            </button>
          </div>
        </form>
      ) : (
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium flex items-center gap-1"
        >
          <Plus className="w-4 h-4" /> New API Key
        </button>
      )}

      {/* Scope reference */}
      <details className="bg-white rounded-xl shadow-sm border border-gray-200">
        <summary className="px-4 py-3 cursor-pointer text-sm font-medium text-gray-700 flex items-center gap-2 select-none">
          <Info className="w-4 h-4 text-blue-500" />
          Scope Reference
        </summary>
        <div className="px-4 pb-4">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-100">
                <th className="py-2 text-left font-medium text-gray-500">Scope</th>
                <th className="py-2 text-left font-medium text-gray-500">Grants Access To</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              <tr>
                <td className="py-2"><code className="text-xs px-1.5 py-0.5 bg-blue-50 text-blue-700 rounded">urls:read</code></td>
                <td className="py-2 text-gray-600">GET short URLs, list/search URLs, view stats, analytics (visits, summaries, by-date, by-field), QR codes</td>
              </tr>
              <tr>
                <td className="py-2"><code className="text-xs px-1.5 py-0.5 bg-amber-50 text-amber-700 rounded">urls:write</code></td>
                <td className="py-2 text-gray-600">Create, update, delete short URLs. Create, rename, delete tags.</td>
              </tr>
              <tr>
                <td className="py-2"><code className="text-xs px-1.5 py-0.5 bg-red-50 text-red-700 rounded">keys:manage</code></td>
                <td className="py-2 text-gray-600">Create, list, roll (rotate), and revoke API keys for your account</td>
              </tr>
            </tbody>
          </table>
        </div>
      </details>

      {/* Key list */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">Loading...</div>
        ) : keys.length === 0 ? (
          <div className="p-8 text-center text-gray-500">No API keys yet.</div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="bg-gray-50 border-b border-gray-200">
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Label</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Key Prefix</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Scopes</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {keys.map((key) => (
                <tr key={key.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm font-medium text-gray-900">{key.label}</td>
                  <td className="px-4 py-3 text-sm font-mono text-gray-500">{key.keyPrefix}...</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {(key.scopes || '').split(',').map((s) => (
                        <span
                          key={s}
                          className="px-1.5 py-0.5 text-xs bg-gray-100 text-gray-600 rounded"
                        >
                          {s.trim()}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {key.createdAt ? new Date(key.createdAt).toLocaleDateString() : '\u2014'}
                  </td>
                  <td className="px-4 py-3">
                    {key.revoked ? (
                      <span className="px-2 py-0.5 text-xs bg-red-100 text-red-700 rounded">Revoked</span>
                    ) : (
                      <span className="px-2 py-0.5 text-xs bg-green-100 text-green-700 rounded">Active</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {!key.revoked && (
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => handleRoll(Number(key.id), key.label)}
                          disabled={rolling === key.id}
                          className="text-gray-400 hover:text-blue-600 disabled:opacity-50"
                          title="Roll (rotate) key — generates a new key with the same label and scopes"
                        >
                          <RefreshCw className={`w-4 h-4 ${rolling === key.id ? 'animate-spin' : ''}`} />
                        </button>
                        <button
                          onClick={() => handleRevoke(Number(key.id))}
                          className="text-gray-400 hover:text-red-600"
                          title="Revoke key permanently"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
