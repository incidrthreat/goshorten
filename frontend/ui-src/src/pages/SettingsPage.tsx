import { useState } from 'react'
import { Info } from 'lucide-react'
import { auth } from '../api/client'

interface SettingsProps {
  user: { email: string; role: string; name: string } | null
  onRefreshUser?: () => void
}

export default function SettingsPage({ user, onRefreshUser }: SettingsProps) {
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
        </dl>
      </div>

      {/* Update Profile */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
          Update Profile
        </h3>
        <form onSubmit={handleUpdateProfile} className="space-y-3">
          {profileError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {profileError}
            </div>
          )}
          {profileSuccess && (
            <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700">
              Profile updated successfully.
            </div>
          )}
          <div>
            <label className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
              Email
            </label>
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

      {/* Change Password */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
          Change Password
        </h3>
        <form onSubmit={handleChangePassword} className="space-y-3">
          {pwError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {pwError}
            </div>
          )}
          {pwSuccess && (
            <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700">
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
