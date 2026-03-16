import { useState } from 'react'
import { Info } from 'lucide-react'

interface SettingsProps {
  user: { email: string; role: string; name: string } | null
}

export default function SettingsPage({ user }: SettingsProps) {
  const [apiBase] = useState(window.location.origin + '/api/v1')

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Account info */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
        <h3 className="text-sm font-medium text-gray-900 mb-4">Account</h3>
        <dl className="space-y-3 text-sm">
          <div className="flex">
            <dt className="w-32 text-gray-500">Email</dt>
            <dd className="text-gray-900">{user?.email || '—'}</dd>
          </div>
          <div className="flex">
            <dt className="w-32 text-gray-500">Name</dt>
            <dd className="text-gray-900">{user?.name || '—'}</dd>
          </div>
          <div className="flex">
            <dt className="w-32 text-gray-500">Role</dt>
            <dd>
              <span
                className={`px-2 py-0.5 text-xs rounded ${
                  user?.role === 'admin'
                    ? 'bg-yellow-100 text-yellow-800'
                    : 'bg-gray-100 text-gray-600'
                }`}
              >
                {user?.role || 'user'}
              </span>
            </dd>
          </div>
        </dl>
      </div>

      {/* API info */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
        <h3 className="text-sm font-medium text-gray-900 mb-4">API</h3>
        <div className="space-y-3 text-sm">
          <div>
            <p className="text-gray-500 mb-1">Base URL</p>
            <code className="px-3 py-1.5 bg-gray-50 border border-gray-200 rounded text-sm font-mono block">
              {apiBase}
            </code>
          </div>
          <div>
            <p className="text-gray-500 mb-1">OpenAPI Spec</p>
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
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
        <h3 className="text-sm font-medium text-gray-900 mb-4">Authentication</h3>
        <div className="space-y-2 text-sm text-gray-600">
          <p>
            <strong>Bearer Token:</strong> Use{' '}
            <code className="px-1 py-0.5 bg-gray-100 rounded text-xs">POST /api/v1/auth/login</code>{' '}
            to obtain a JWT, then send it as{' '}
            <code className="px-1 py-0.5 bg-gray-100 rounded text-xs">
              Authorization: Bearer &lt;token&gt;
            </code>
          </p>
          <p>
            <strong>API Key:</strong> Create a key via the API Keys page, then send as{' '}
            <code className="px-1 py-0.5 bg-gray-100 rounded text-xs">
              Authorization: ApiKey &lt;key&gt;
            </code>
          </p>
          <p>
            <strong>OIDC:</strong> If configured, use the{' '}
            <code className="px-1 py-0.5 bg-gray-100 rounded text-xs">/api/v1/auth/oidc/</code>{' '}
            endpoints for SSO login flows.
          </p>
        </div>
      </div>

      {/* Version */}
      <div className="flex items-center gap-2 text-xs text-gray-400">
        <Info className="w-3.5 h-3.5" />
        GoShorten v0.5.0 &middot; Phase 7 Frontend
      </div>
    </div>
  )
}
