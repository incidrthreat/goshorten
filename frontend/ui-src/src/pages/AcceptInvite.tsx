import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { BarChart3, Building2 } from 'lucide-react'
import { invitations, type InvitePreview } from '../api/client'

const ROLE_LABELS: Record<string, string> = {
  admin: 'Admin',
  member: 'Member',
  viewer: 'Viewer (read-only)',
}

export default function AcceptInvite() {
  const { token = '' } = useParams()
  const loggedIn = !!localStorage.getItem('token')

  const [preview, setPreview] = useState<InvitePreview | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')

  useEffect(() => {
    invitations
      .preview(token)
      .then(setPreview)
      .catch((err) => setError(err instanceof Error ? err.message : 'Invitation not found'))
      .finally(() => setLoading(false))
  }, [token])

  const finish = (workspaceId: number) => {
    localStorage.setItem('workspaceId', String(workspaceId))
    window.location.href = '/'
  }

  const acceptExisting = async () => {
    setBusy(true)
    setError('')
    try {
      const res = await invitations.accept(token)
      finish(res.workspaceId)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to accept invitation')
      setBusy(false)
    }
  }

  const acceptWithSignup = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const res = await invitations.accept(token, { name: name.trim(), password })
      if (res.token) localStorage.setItem('token', res.token)
      finish(res.workspaceId)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create account')
      setBusy(false)
    }
  }

  const goToLogin = () => {
    // Preserve the invite so the user can reopen it after signing in.
    sessionStorage.setItem('pendingInvite', token)
    window.location.href = '/'
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-900 px-4">
      <div className="w-full max-w-md">
        <BarChart3 className="w-12 h-12 text-blue-400 mx-auto mb-4" />
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-lg p-6">
          {loading ? (
            <p className="text-center text-gray-500 text-sm">Loading invitation...</p>
          ) : error && !preview ? (
            <div className="text-center">
              <p className="text-sm font-medium text-red-700 mb-4">{error}</p>
              <a href="/" className="inline-block px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700">
                Go to GoShorten
              </a>
            </div>
          ) : preview ? (
            <div className="space-y-4">
              <div className="text-center">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 text-sm font-medium">
                  <Building2 className="w-4 h-4" />
                  {preview.workspaceName}
                </div>
                <h2 className="mt-3 text-lg font-semibold text-gray-900 dark:text-gray-100">
                  You're invited to join
                </h2>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  As <strong>{ROLE_LABELS[preview.role] || preview.role}</strong>, for{' '}
                  <strong>{preview.email}</strong>.
                </p>
              </div>

              {error && (
                <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
                  {error}
                </div>
              )}

              {loggedIn ? (
                <button
                  onClick={acceptExisting}
                  disabled={busy}
                  className="w-full px-4 py-2.5 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50"
                >
                  {busy ? 'Joining...' : 'Accept invitation'}
                </button>
              ) : preview.hasAccount ? (
                <div className="space-y-3">
                  <p className="text-sm text-gray-600 dark:text-gray-400 text-center">
                    You already have an account. Sign in to accept this invitation.
                  </p>
                  <button
                    onClick={goToLogin}
                    className="w-full px-4 py-2.5 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700"
                  >
                    Sign in
                  </button>
                </div>
              ) : preview.passwordLoginEnabled ? (
                <form onSubmit={acceptWithSignup} className="space-y-3">
                  <p className="text-sm text-gray-600 dark:text-gray-400 text-center">
                    Create your account to join.
                  </p>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Display name</label>
                    <input
                      type="text"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
                      Password <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="password"
                      required
                      minLength={8}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 outline-none text-sm"
                    />
                  </div>
                  <button
                    type="submit"
                    disabled={busy}
                    className="w-full px-4 py-2.5 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50"
                  >
                    {busy ? 'Creating account...' : 'Create account & join'}
                  </button>
                </form>
              ) : (
                <p className="text-sm text-gray-600 dark:text-gray-400 text-center">
                  Password signup is disabled on this instance. Please ask an administrator to
                  create your account, then reopen this invitation.
                </p>
              )}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}
