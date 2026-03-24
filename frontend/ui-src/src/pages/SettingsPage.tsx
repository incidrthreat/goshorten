import { useState, useEffect } from 'react'
import { Info, Monitor, Sun, Moon, Laptop, Shield, LogOut, Link } from 'lucide-react'
import { auth } from '../api/client'

interface Session {
  id: string
  label: string
  ipAddress: string
  createdAt: string
  expiresAt: string
  isCurrent: boolean
}

interface SignInEvent {
  ipAddress: string
  userAgent: string
  success: boolean
  signedInAt: string
}

interface SettingsProps {
  user: { email: string; role: string; name: string } | null
  onRefreshUser?: () => void
  isOIDC?: boolean
  theme: string
  onSetTheme: (t: string) => void
}

export default function SettingsPage({ user, onRefreshUser, isOIDC, theme, onSetTheme }: SettingsProps) {
  const [apiBase] = useState(window.location.origin + '/api/v1')

  // --- Update Profile ---
  const [profileForm, setProfileForm] = useState({
    email: user?.email || '',
    name: user?.name || '',
  })
  const [profileLoading, setProfileLoading] = useState(false)
  const [profileError, setProfileError] = useState('')
  const [profileSuccess, setProfileSuccess] = useState(false)

  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault()
    setProfileError('')
    setProfileSuccess(false)
    setProfileLoading(true)
    try {
      await auth.updateProfile({
        email: profileForm.email || undefined,
        name: profileForm.name || undefined,
      })
      setProfileSuccess(true)
      onRefreshUser?.()
    } catch (err) {
      setProfileError(err instanceof Error ? err.message : 'Failed to update profile')
    } finally {
      setProfileLoading(false)
    }
  }

  // --- Change Password ---
  const [pwForm, setPwForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })
  const [pwLoading, setPwLoading] = useState(false)
  const [pwError, setPwError] = useState('')
  const [pwSuccess, setPwSuccess] = useState(false)

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    setPwError('')
    setPwSuccess(false)

    if (pwForm.newPassword !== pwForm.confirmPassword) {
      setPwError('New passwords do not match')
      return
    }
    if (pwForm.newPassword.length < 8) {
      setPwError('New password must be at least 8 characters')
      return
    }

    setPwLoading(true)
    try {
      await auth.changePassword(pwForm.currentPassword, pwForm.newPassword)
      setPwSuccess(true)
      setPwForm({ currentPassword: '', newPassword: '', confirmPassword: '' })
    } catch (err) {
      setPwError(err instanceof Error ? err.message : 'Failed to change password')
    } finally {
      setPwLoading(false)
    }
  }

  // --- Sessions ---
  const [sessions, setSessions] = useState<Session[]>([])
  const [sessionsLoading, setSessionsLoading] = useState(true)
  const [revokeAllLoading, setRevokeAllLoading] = useState(false)
  const [revokingId, setRevokingId] = useState<string | null>(null)

  const loadSessions = async () => {
    try {
      const res = await auth.sessions.list()
      setSessions(res.sessions)
    } catch {
      // non-fatal
    } finally {
      setSessionsLoading(false)
    }
  }

  useEffect(() => {
    loadSessions()
  }, [])

  const handleRevokeSession = async (id: string) => {
    setRevokingId(id)
    try {
      await auth.sessions.revoke(id)
      setSessions((prev) => prev.filter((s) => s.id !== id))
    } catch {
      // ignore
    } finally {
      setRevokingId(null)
    }
  }

  const handleRevokeAll = async () => {
    if (!confirm('Sign out all other devices?')) return
    setRevokeAllLoading(true)
    try {
      await auth.sessions.revokeAll()
      await loadSessions()
    } catch {
      // ignore
    } finally {
      setRevokeAllLoading(false)
    }
  }

  // --- Sign-in History ---
  const [signInHistory, setSignInHistory] = useState<SignInEvent[]>([])
  const [historyLoading, setHistoryLoading] = useState(true)

  useEffect(() => {
    auth.signInHistory()
      .then((res) => setSignInHistory(res.events))
      .catch(() => {})
      .finally(() => setHistoryLoading(false))
  }, [])

  const themeOptions = [
    { value: 'system', label: 'System', icon: Laptop },
    { value: 'light', label: 'Light', icon: Sun },
    { value: 'dark', label: 'Dark', icon: Moon },
  ]

  const formatDate = (iso: string) => {
    try {
      return new Date(iso).toLocaleString()
    } catch {
      return iso
    }
  }

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Account info */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">Account</h3>
        <dl className="space-y-3 text-sm">
          <div className="flex">
            <dt className="w-32 text-gray-500 dark:text-gray-400">Email</dt>
            <dd className="text-gray-900 dark:text-gray-100">{user?.email || '—'}</dd>
          </div>
          <div className="flex">
            <dt className="w-32 text-gray-500 dark:text-gray-400">Name</dt>
            <dd className="text-gray-900 dark:text-gray-100">{user?.name || '—'}</dd>
          </div>
          <div className="flex">
            <dt className="w-32 text-gray-500 dark:text-gray-400">Role</dt>
            <dd>
              <span
                className={`px-2 py-0.5 text-xs rounded ${
                  user?.role === 'admin'
                    ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                    : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
                }`}
              >
                {user?.role || 'user'}
              </span>
            </dd>
          </div>
          {isOIDC && (
            <div className="flex">
              <dt className="w-32 text-gray-500 dark:text-gray-400">Login</dt>
              <dd>
                <span className="px-2 py-0.5 text-xs rounded bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                  SSO / OIDC
                </span>
              </dd>
            </div>
          )}
        </dl>
      </div>

      {/* Update Profile — hidden for OIDC users */}
      {!isOIDC && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
            Update Profile
          </h3>
          <form onSubmit={handleUpdateProfile} className="space-y-3">
            {profileError && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700 dark:bg-red-900/30 dark:border-red-800 dark:text-red-300">
                {profileError}
              </div>
            )}
            {profileSuccess && (
              <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700 dark:bg-green-900/30 dark:border-green-800 dark:text-green-300">
                Profile updated successfully.
              </div>
            )}
            <div>
              <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">Email</label>
              <input
                type="email"
                value={profileForm.email}
                onChange={(e) => setProfileForm({ ...profileForm, email: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                Display Name
              </label>
              <input
                type="text"
                value={profileForm.name}
                onChange={(e) => setProfileForm({ ...profileForm, name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
              />
            </div>
            <button
              type="submit"
              disabled={profileLoading}
              className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {profileLoading ? 'Saving...' : 'Update Profile'}
            </button>
          </form>
        </div>
      )}

      {/* Change Password — hidden for OIDC users */}
      {!isOIDC && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
            Change Password
          </h3>
          <form onSubmit={handleChangePassword} className="space-y-3">
            {pwError && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700 dark:bg-red-900/30 dark:border-red-800 dark:text-red-300">
                {pwError}
              </div>
            )}
            {pwSuccess && (
              <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700 dark:bg-green-900/30 dark:border-green-800 dark:text-green-300">
                Password changed successfully.
              </div>
            )}
            <div>
              <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                Current Password
              </label>
              <input
                type="password"
                value={pwForm.currentPassword}
                onChange={(e) => setPwForm({ ...pwForm, currentPassword: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                New Password
              </label>
              <input
                type="password"
                value={pwForm.newPassword}
                onChange={(e) => setPwForm({ ...pwForm, newPassword: e.target.value })}
                required
                minLength={8}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                Confirm New Password
              </label>
              <input
                type="password"
                value={pwForm.confirmPassword}
                onChange={(e) => setPwForm({ ...pwForm, confirmPassword: e.target.value })}
                required
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
              />
            </div>
            <button
              type="submit"
              disabled={pwLoading}
              className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {pwLoading ? 'Saving...' : 'Update Password'}
            </button>
          </form>
        </div>
      )}

      {/* Appearance (12.4) */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4 flex items-center gap-2">
          <Monitor className="w-4 h-4" />
          Appearance
        </h3>
        <div className="flex gap-3">
          {themeOptions.map(({ value, label, icon: Icon }) => (
            <button
              key={value}
              onClick={() => onSetTheme(value)}
              className={`flex-1 flex flex-col items-center gap-2 py-3 px-2 rounded-lg border text-sm font-medium transition-colors ${
                theme === value
                  ? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300 dark:border-blue-500'
                  : 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:border-gray-300 dark:hover:border-gray-600'
              }`}
            >
              <Icon className="w-5 h-5" />
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Security — Sessions (12.2) */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
            <Shield className="w-4 h-4" />
            Active Sessions
          </h3>
          <button
            onClick={handleRevokeAll}
            disabled={revokeAllLoading}
            className="text-xs px-3 py-1.5 border border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50 transition-colors flex items-center gap-1.5"
          >
            <LogOut className="w-3 h-3" />
            {revokeAllLoading ? 'Signing out...' : 'Sign out other devices'}
          </button>
        </div>
        {sessionsLoading ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading sessions...</p>
        ) : sessions.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">No active sessions found.</p>
        ) : (
          <div className="space-y-2">
            {sessions.map((s) => (
              <div
                key={s.id}
                className="flex items-center justify-between p-3 rounded-lg bg-gray-50 dark:bg-gray-800 border border-gray-100 dark:border-gray-700"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                      {s.label || 'Unknown device'}
                    </span>
                    {s.isCurrent && (
                      <span className="px-1.5 py-0.5 text-xs bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300 rounded">
                        current
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                    {s.ipAddress && <span>{s.ipAddress} · </span>}
                    <span>Active since {formatDate(s.createdAt)}</span>
                  </div>
                </div>
                {!s.isCurrent && (
                  <button
                    onClick={() => handleRevokeSession(s.id)}
                    disabled={revokingId === s.id}
                    className="ml-3 text-xs px-2.5 py-1 text-red-600 dark:text-red-400 border border-red-200 dark:border-red-800 rounded hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50 transition-colors shrink-0"
                  >
                    {revokingId === s.id ? '...' : 'Revoke'}
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Sign-in History (12.2) */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
          Recent Sign-in History
        </h3>
        {historyLoading ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading...</p>
        ) : signInHistory.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">No sign-in history.</p>
        ) : (
          <div className="space-y-1.5">
            {signInHistory.map((e, i) => (
              <div
                key={i}
                className="flex items-center gap-3 p-2.5 rounded-lg bg-gray-50 dark:bg-gray-800 text-sm"
              >
                <span
                  className={`shrink-0 w-2 h-2 rounded-full ${
                    e.success ? 'bg-green-500' : 'bg-red-500'
                  }`}
                />
                <span className="text-gray-500 dark:text-gray-400 shrink-0 text-xs">
                  {formatDate(e.signedInAt)}
                </span>
                <span className="text-gray-700 dark:text-gray-300 truncate text-xs">
                  {e.ipAddress || '—'} · {e.userAgent ? e.userAgent.slice(0, 60) : 'Unknown'}
                </span>
                {!e.success && (
                  <span className="shrink-0 text-xs text-red-500 dark:text-red-400">Failed</span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Admin section (12.6) — admin only */}
      {user?.role === 'admin' && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
            Administration
          </h3>
          <div className="grid grid-cols-2 gap-3">
            <a
              href="/admin/users"
              className="flex items-center gap-2 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-blue-400 dark:hover:border-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/10 transition-colors text-sm text-gray-700 dark:text-gray-300"
            >
              <Link className="w-4 h-4 text-blue-500" />
              User Management
            </a>
            <a
              href="/admin/oidc-providers"
              className="flex items-center gap-2 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-blue-400 dark:hover:border-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/10 transition-colors text-sm text-gray-700 dark:text-gray-300"
            >
              <Shield className="w-4 h-4 text-blue-500" />
              SSO / OIDC Providers
            </a>
          </div>
        </div>
      )}

      {/* API info */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">API</h3>
        <div className="space-y-3 text-sm">
          <div>
            <p className="text-gray-500 dark:text-gray-400 mb-1">Base URL</p>
            <code className="px-3 py-1.5 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded text-sm font-mono block text-gray-800 dark:text-gray-200">
              {apiBase}
            </code>
          </div>
          <div>
            <p className="text-gray-500 dark:text-gray-400 mb-1">OpenAPI Spec</p>
            <a
              href="/api/v1/swagger.json"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-600 hover:underline"
            >
              /api/v1/swagger.json
            </a>
          </div>
        </div>
      </div>

      {/* Auth methods */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
          Authentication
        </h3>
        <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
          <p>
            <strong className="text-gray-900 dark:text-gray-200">Bearer Token:</strong> Use{' '}
            <code className="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-xs">
              POST /api/v1/auth/login
            </code>{' '}
            to obtain a JWT, then send it as{' '}
            <code className="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-xs">
              Authorization: Bearer &lt;token&gt;
            </code>
          </p>
          <p>
            <strong className="text-gray-900 dark:text-gray-200">API Key:</strong> Create a key via
            the API Keys page, then send as{' '}
            <code className="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-xs">
              Authorization: ApiKey &lt;key&gt;
            </code>
          </p>
          <p>
            <strong className="text-gray-900 dark:text-gray-200">OIDC:</strong> If configured, use
            the{' '}
            <code className="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-xs">
              /api/v1/auth/oidc/
            </code>{' '}
            endpoints for SSO login flows.
          </p>
        </div>
      </div>

      {/* Version */}
      <div className="flex items-center gap-2 text-xs text-gray-400">
        <Info className="w-3.5 h-3.5" />
        GoShorten &middot; v{import.meta.env.VITE_APP_VERSION || 'dev'}
      </div>
    </div>
  )
}
