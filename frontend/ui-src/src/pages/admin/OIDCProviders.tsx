import { useState, useEffect } from 'react'
import { Plus, Trash2, Edit2, X, Check, Lock, AlertTriangle } from 'lucide-react'
import { admin } from '../../api/client'

interface Provider {
  id: number
  name: string
  issuerUrl: string
  clientId: string
  redirectUri: string
  scopes: string
  isEnabled: boolean
  autoRegister: boolean
  defaultRole: string
}

interface CreateForm {
  name: string
  issuerUrl: string
  clientId: string
  clientSecret: string
  redirectUri: string
  scopes: string
  autoRegister: boolean
  defaultRole: string
  isEnabled: boolean
}

const emptyForm = (): CreateForm => ({
  name: '',
  issuerUrl: '',
  clientId: '',
  clientSecret: '',
  redirectUri: '',
  scopes: 'openid email profile',
  autoRegister: true,
  defaultRole: 'user',
  isEnabled: true,
})

export default function AdminOIDCProviders() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Auth settings
  const [passwordLoginEnabled, setPasswordLoginEnabled] = useState(true)
  const [envOverride, setEnvOverride] = useState(false)
  const [settingsLoading, setSettingsLoading] = useState(true)
  const [settingsSaving, setSettingsSaving] = useState(false)

  const [showCreate, setShowCreate] = useState(false)
  const [createForm, setCreateForm] = useState<CreateForm>(emptyForm())
  const [createLoading, setCreateLoading] = useState(false)
  const [createError, setCreateError] = useState('')

  const [editingName, setEditingName] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<Partial<Provider & { clientSecret: string }>>({})
  const [editLoading, setEditLoading] = useState(false)
  const [deletingName, setDeletingName] = useState<string | null>(null)

  const load = async () => {
    try {
      const res = await admin.oidcProviders.list()
      setProviders(res.providers)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load providers')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    admin.settings.get()
      .then((s) => {
        setPasswordLoginEnabled(s.passwordLoginEnabled)
        setEnvOverride(s.envOverride)
      })
      .catch(() => {})
      .finally(() => setSettingsLoading(false))
  }, [])

  const handleTogglePasswordLogin = async (enabled: boolean) => {
    // Guard: don't allow disabling password login if no OIDC providers are enabled
    if (!enabled) {
      const enabledProviders = providers.filter((p) => p.isEnabled)
      if (enabledProviders.length === 0) {
        if (!confirm(
          'Warning: You have no enabled OIDC providers.\n\n' +
          'Disabling password login will lock all users out of the system.\n\n' +
          'Are you sure you want to continue?'
        )) return
      }
    }
    setSettingsSaving(true)
    try {
      await admin.settings.update({ passwordLoginEnabled: enabled })
      setPasswordLoginEnabled(enabled)
    } catch {
      // ignore
    } finally {
      setSettingsSaving(false)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreateError('')
    setCreateLoading(true)
    try {
      await admin.oidcProviders.create({
        name: createForm.name,
        issuerUrl: createForm.issuerUrl,
        clientId: createForm.clientId,
        clientSecret: createForm.clientSecret || undefined,
        redirectUri: createForm.redirectUri || undefined,
        scopes: createForm.scopes || undefined,
        autoRegister: createForm.autoRegister,
        defaultRole: createForm.defaultRole || undefined,
      })
      setShowCreate(false)
      setCreateForm(emptyForm())
      await load()
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Failed to create provider')
    } finally {
      setCreateLoading(false)
    }
  }

  const handleEdit = async (name: string) => {
    setEditLoading(true)
    try {
      await admin.oidcProviders.update(name, {
        isEnabled: editForm.isEnabled,
        autoRegister: editForm.autoRegister,
        defaultRole: editForm.defaultRole,
        clientSecret: editForm.clientSecret || undefined,
      })
      setEditingName(null)
      await load()
    } catch {
      // ignore
    } finally {
      setEditLoading(false)
    }
  }

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete OIDC provider "${name}"? This cannot be undone.`)) return
    setDeletingName(name)
    try {
      await admin.oidcProviders.delete(name)
      setProviders((prev) => prev.filter((p) => p.name !== name))
    } catch {
      // ignore
    } finally {
      setDeletingName(null)
    }
  }

  const startEdit = (p: Provider) => {
    setEditingName(p.name)
    setEditForm({
      isEnabled: p.isEnabled,
      autoRegister: p.autoRegister,
      defaultRole: p.defaultRole,
      clientSecret: '',
    })
  }

  const inputCls =
    'w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm'

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">SSO / OIDC Providers</h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
            Configure external identity providers for single sign-on.
          </p>
        </div>
        <button
          onClick={() => { setShowCreate(true); setCreateError('') }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus className="w-4 h-4" />
          Add Provider
        </button>
      </div>

      {/* Auth settings */}
      {!settingsLoading && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-5">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Authentication Settings</h3>
          <div className="flex items-center justify-between">
            <div className="min-w-0">
              <p className="text-sm text-gray-700 dark:text-gray-300">Password login</p>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                Allow users to sign in with email and password.
              </p>
              {envOverride && (
                <p className="text-xs text-amber-600 dark:text-amber-400 mt-1 flex items-center gap-1">
                  <Lock className="w-3 h-3" />
                  Controlled by GOSHORTEN_DISABLE_PASSWORD_LOGIN environment variable.
                </p>
              )}
            </div>
            <button
              type="button"
              disabled={envOverride || settingsSaving}
              onClick={() => handleTogglePasswordLogin(!passwordLoginEnabled)}
              className={`relative inline-flex h-6 w-11 shrink-0 rounded-full border-2 border-transparent transition-colors disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none ${
                passwordLoginEnabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
              }`}
              role="switch"
              aria-checked={passwordLoginEnabled}
            >
              <span
                className={`inline-block h-5 w-5 rounded-full bg-white shadow transform transition-transform ${
                  passwordLoginEnabled ? 'translate-x-5' : 'translate-x-0'
                }`}
              />
            </button>
          </div>
        </div>
      )}

      {/* Lockout warning: password disabled + no enabled OIDC providers */}
      {!settingsLoading && !passwordLoginEnabled && providers.filter((p) => p.isEnabled).length === 0 && (
        <div className="flex items-start gap-3 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-xl text-sm">
          <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
          <div>
            <p className="font-medium text-amber-800 dark:text-amber-300">Users are locked out</p>
            <p className="text-amber-700 dark:text-amber-400 mt-0.5">
              Password login is disabled and no OIDC providers are enabled. Nobody can sign in.
              Enable at least one OIDC provider below, or re-enable password login above.
            </p>
          </div>
        </div>
      )}

      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-xl text-sm text-red-700 dark:text-red-300">
          {error}
        </div>
      )}

      {/* Create form */}
      {showCreate && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-blue-200 dark:border-blue-800 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">New OIDC Provider</h3>
            <button onClick={() => setShowCreate(false)}>
              <X className="w-4 h-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200" />
            </button>
          </div>
          <form onSubmit={handleCreate} className="space-y-3">
            {createError && (
              <div className="p-3 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-300">
                {createError}
              </div>
            )}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Provider Name *</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. google, okta"
                  value={createForm.name}
                  onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                  className={inputCls}
                />
              </div>
              <div>
                <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Issuer URL *</label>
                <input
                  type="url"
                  required
                  placeholder="https://accounts.google.com"
                  value={createForm.issuerUrl}
                  onChange={(e) => setCreateForm({ ...createForm, issuerUrl: e.target.value })}
                  className={inputCls}
                />
              </div>
              <div>
                <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Client ID *</label>
                <input
                  type="text"
                  required
                  value={createForm.clientId}
                  onChange={(e) => setCreateForm({ ...createForm, clientId: e.target.value })}
                  className={inputCls}
                />
              </div>
              <div>
                <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Client Secret</label>
                <input
                  type="password"
                  value={createForm.clientSecret}
                  onChange={(e) => setCreateForm({ ...createForm, clientSecret: e.target.value })}
                  className={inputCls}
                />
              </div>
              <div>
                <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Redirect URI</label>
                <input
                  type="url"
                  placeholder={window.location.origin + '/auth/callback'}
                  value={createForm.redirectUri}
                  onChange={(e) => setCreateForm({ ...createForm, redirectUri: e.target.value })}
                  className={inputCls}
                />
              </div>
              <div>
                <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Scopes</label>
                <input
                  type="text"
                  value={createForm.scopes}
                  onChange={(e) => setCreateForm({ ...createForm, scopes: e.target.value })}
                  className={inputCls}
                />
              </div>
              <div>
                <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Default Role</label>
                <select
                  value={createForm.defaultRole}
                  onChange={(e) => setCreateForm({ ...createForm, defaultRole: e.target.value })}
                  className={inputCls}
                >
                  <option value="user">user</option>
                  <option value="admin">admin</option>
                </select>
              </div>
              <div className="flex flex-col justify-end gap-2">
                <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={createForm.autoRegister}
                    onChange={(e) => setCreateForm({ ...createForm, autoRegister: e.target.checked })}
                    className="rounded border-gray-300 dark:border-gray-600"
                  />
                  Auto-register new users
                </label>
                <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={createForm.isEnabled}
                    onChange={(e) => setCreateForm({ ...createForm, isEnabled: e.target.checked })}
                    className="rounded border-gray-300 dark:border-gray-600"
                  />
                  Enabled
                </label>
              </div>
            </div>
            <div className="flex gap-2 pt-1">
              <button
                type="submit"
                disabled={createLoading}
                className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {createLoading ? 'Creating...' : 'Create Provider'}
              </button>
              <button
                type="button"
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Providers list */}
      {loading ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">Loading providers...</p>
      ) : providers.length === 0 ? (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-8 text-center">
          <p className="text-sm text-gray-500 dark:text-gray-400">No OIDC providers configured.</p>
          <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
            Add a provider to enable single sign-on.
          </p>
        </div>
      ) : (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-100 dark:border-gray-800">
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                  Name
                </th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                  Issuer
                </th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                  Default Role
                </th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                  Status
                </th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
              {providers.map((p) => (
                <tr key={p.name}>
                  {editingName === p.name ? (
                    <td colSpan={5} className="px-4 py-3">
                      <div className="flex items-center gap-3 flex-wrap">
                        <label className="flex items-center gap-1.5 text-xs text-gray-700 dark:text-gray-300">
                          <input
                            type="checkbox"
                            checked={editForm.isEnabled}
                            onChange={(e) => setEditForm({ ...editForm, isEnabled: e.target.checked })}
                          />
                          Enabled
                        </label>
                        <label className="flex items-center gap-1.5 text-xs text-gray-700 dark:text-gray-300">
                          <input
                            type="checkbox"
                            checked={editForm.autoRegister}
                            onChange={(e) => setEditForm({ ...editForm, autoRegister: e.target.checked })}
                          />
                          Auto-register
                        </label>
                        <select
                          value={editForm.defaultRole}
                          onChange={(e) => setEditForm({ ...editForm, defaultRole: e.target.value })}
                          className="px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                        >
                          <option value="user">user</option>
                          <option value="admin">admin</option>
                        </select>
                        <input
                          type="password"
                          placeholder="New client secret (leave blank to keep)"
                          value={editForm.clientSecret}
                          onChange={(e) => setEditForm({ ...editForm, clientSecret: e.target.value })}
                          className="px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 w-64"
                        />
                        <button
                          onClick={() => handleEdit(p.name)}
                          disabled={editLoading}
                          className="p-1.5 text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20 rounded disabled:opacity-50"
                        >
                          <Check className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => setEditingName(null)}
                          className="p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  ) : (
                    <>
                      <td className="px-4 py-3 font-medium text-gray-900 dark:text-gray-100">
                        {p.name}
                      </td>
                      <td className="px-4 py-3 text-gray-500 dark:text-gray-400 truncate max-w-xs">
                        {p.issuerUrl}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-0.5 text-xs rounded ${
                          p.defaultRole === 'admin'
                            ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300'
                            : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-300'
                        }`}>
                          {p.defaultRole}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <span className={`w-2 h-2 rounded-full ${p.isEnabled ? 'bg-green-500' : 'bg-gray-400'}`} />
                          <span className="text-xs text-gray-500 dark:text-gray-400">
                            {p.isEnabled ? 'Enabled' : 'Disabled'}
                          </span>
                          {p.autoRegister && (
                            <span className="px-1.5 py-0.5 text-xs bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300 rounded">
                              auto-register
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1 justify-end">
                          <button
                            onClick={() => startEdit(p)}
                            className="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors"
                            title="Edit"
                          >
                            <Edit2 className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => handleDelete(p.name)}
                            disabled={deletingName === p.name}
                            className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors disabled:opacity-50"
                            title="Delete"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
