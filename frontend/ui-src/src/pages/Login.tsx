import { useState, useEffect } from 'react'
import { BarChart3, LogIn, AlertTriangle, RefreshCw } from 'lucide-react'
import { auth } from '../api/client'

interface OIDCProvider {
  id: number
  name: string
  isEnabled: boolean
}

interface LoginProps {
  onLogin: (email: string, password: string) => Promise<void>
}

export default function Login({ onLogin }: LoginProps) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const [passwordEnabled, setPasswordEnabled] = useState(true)
  const [oidcProviders, setOidcProviders] = useState<OIDCProvider[]>([])
  const [configLoading, setConfigLoading] = useState(true)
  const [oidcLoading, setOidcLoading] = useState<string | null>(null)
  const [fetchError, setFetchError] = useState('')

  const loadLoginMethods = () => {
    setFetchError('')
    setConfigLoading(true)

    Promise.all([
      auth.config(),
      auth.oidc.providers(),
    ])
      .then(([cfg, prov]) => {
        setPasswordEnabled(cfg.passwordLoginEnabled)
        setOidcProviders(prov.providers.filter((p) => p.isEnabled))
      })
      .catch((err) => {
        // One or both calls failed — fall back to password login so the user
        // isn't locked out, but surface the error.
        setPasswordEnabled(true)
        setFetchError(err instanceof Error ? err.message : 'Could not load login configuration. Showing password login as fallback.')
      })
      .finally(() => setConfigLoading(false))
  }

  useEffect(() => {
    loadLoginMethods()
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await onLogin(email, password)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  const handleOIDCLogin = async (providerName: string) => {
    setError('')
    setOidcLoading(providerName)
    try {
      const res = await auth.oidc.authUrl(providerName)
      sessionStorage.setItem('oidc_pending', JSON.stringify({ provider: providerName, state: res.state }))
      window.location.href = res.authUrl
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start SSO login')
      setOidcLoading(null)
    }
  }

  const showDivider = oidcProviders.length > 0 && passwordEnabled
  const noMethods = !passwordEnabled && oidcProviders.length === 0 && !fetchError

  if (configLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-900">
        <div className="text-gray-400 text-sm">Loading...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-900 px-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <BarChart3 className="w-12 h-12 text-blue-400 mx-auto mb-3" />
          <h1 className="text-2xl font-bold text-white">GoShorten</h1>
          <p className="text-gray-400 mt-1">Sign in to your account</p>
        </div>

        <div className="bg-white rounded-xl shadow-lg p-6 space-y-4">
          {/* Config fetch error — non-fatal, fell back to password */}
          {fetchError && (
            <div className="flex items-start gap-2 p-3 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-700">
              <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
              <div className="flex-1 min-w-0">
                <p>{fetchError}</p>
                <button
                  onClick={loadLoginMethods}
                  className="mt-1 flex items-center gap-1 text-xs text-amber-600 hover:text-amber-800 underline"
                >
                  <RefreshCw className="w-3 h-3" /> Retry
                </button>
              </div>
            </div>
          )}

          {/* Login error */}
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {error}
            </div>
          )}

          {/* OIDC provider buttons */}
          {oidcProviders.length > 0 && (
            <div className="space-y-2">
              {oidcProviders.map((p) => (
                <button
                  key={p.name}
                  type="button"
                  disabled={oidcLoading !== null}
                  onClick={() => handleOIDCLogin(p.name)}
                  className="w-full flex items-center justify-center gap-2 py-2.5 px-4 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 transition-colors"
                >
                  <LogIn className="w-4 h-4" />
                  {oidcLoading === p.name
                    ? 'Redirecting...'
                    : `Sign in with ${p.name.charAt(0).toUpperCase() + p.name.slice(1)}`}
                </button>
              ))}
            </div>
          )}

          {/* Divider */}
          {showDivider && (
            <div className="flex items-center gap-3">
              <div className="flex-1 h-px bg-gray-200" />
              <span className="text-xs text-gray-400">or</span>
              <div className="flex-1 h-px bg-gray-200" />
            </div>
          )}

          {/* Password login form */}
          {passwordEnabled && (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoComplete="email"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                  placeholder="admin@goshorten.local"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  autoComplete="current-password"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                />
              </div>
              <button
                type="submit"
                disabled={loading}
                className="w-full py-2.5 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {loading ? 'Signing in...' : 'Sign in'}
              </button>
            </form>
          )}

          {/* Password is disabled + no OIDC providers enabled */}
          {noMethods && (
            <div className="text-center space-y-3">
              <p className="text-sm text-gray-500">
                No login methods are currently enabled.
              </p>
              <p className="text-xs text-gray-400">
                An administrator must enable at least one OIDC provider or re-enable
                password login before anyone can sign in.
              </p>
              <button
                onClick={loadLoginMethods}
                className="flex items-center gap-1.5 mx-auto text-xs text-blue-600 hover:text-blue-800 underline"
              >
                <RefreshCw className="w-3 h-3" /> Check again
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
