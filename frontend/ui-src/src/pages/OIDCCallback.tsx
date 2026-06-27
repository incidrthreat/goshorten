import { useEffect, useState } from 'react'
import { BarChart3 } from 'lucide-react'
import { auth } from '../api/client'

export default function OIDCCallback() {
  const [error, setError] = useState('')

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const code = params.get('code')
    const state = params.get('state')
    const errorParam = params.get('error')
    const errorDesc = params.get('error_description')

    if (errorParam) {
      setError(errorDesc || errorParam)
      return
    }

    if (!code || !state) {
      setError('Missing code or state parameter in callback URL.')
      return
    }

    const raw = sessionStorage.getItem('oidc_pending')
    if (!raw) {
      setError('No pending SSO login found. Please try again.')
      return
    }

    let pending: { provider: string; state: string }
    try {
      pending = JSON.parse(raw)
    } catch {
      setError('Invalid SSO session data. Please try again.')
      return
    }

    if (pending.state !== state) {
      setError('State mismatch. Possible CSRF attack — please try logging in again.')
      return
    }

    sessionStorage.removeItem('oidc_pending')

    auth.oidc.callback(pending.provider, code, state)
      .then((res) => {
        localStorage.setItem('token', res.token)
        window.location.replace('/')
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'SSO login failed. Please try again.')
      })
  }, [])

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-900 px-4">
      <div className="w-full max-w-sm text-center">
        <BarChart3 className="w-12 h-12 text-blue-400 mx-auto mb-4" />
        {error ? (
          <div className="bg-white rounded-xl shadow-lg p-6">
            <p className="text-sm font-medium text-red-700 mb-4">{error}</p>
            <a
              href="/"
              className="inline-block px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors"
            >
              Back to login
            </a>
          </div>
        ) : (
          <div className="text-gray-400 text-sm">Completing sign-in...</div>
        )}
      </div>
    </div>
  )
}
