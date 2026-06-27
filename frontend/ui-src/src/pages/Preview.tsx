import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'

const API_BASE = '/api/v1'

interface PreviewData {
  code: string
  longUrl: string
  title: string
  createdAt: string
  totalClicks: string
  isActive: boolean
  domain: string
  tags: string[]
}

export default function Preview() {
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()
  const [data, setData] = useState<PreviewData | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!code) return
    fetch(`${API_BASE}/short-urls/${code}/preview`)
      .then(async (res) => {
        if (!res.ok) throw new Error('Link not found')
        return res.json()
      })
      .then(setData)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [code])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-900">
        <div className="text-gray-400">Loading preview...</div>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-900">
        <div className="max-w-md w-full bg-white rounded-2xl shadow-xl p-8 text-center">
          <div className="text-5xl mb-4">🔗</div>
          <h1 className="text-2xl font-bold text-gray-800 mb-2">Link Not Found</h1>
          <p className="text-gray-500 mb-6">
            The short link <span className="font-mono font-semibold">{code}</span> does not exist or has expired.
          </p>
          <button
            onClick={() => navigate('/')}
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
          >
            Go Home
          </button>
        </div>
      </div>
    )
  }

  const createdDate = data.createdAt
    ? new Date(data.createdAt).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      })
    : 'Unknown'

  const destinationHost = (() => {
    try {
      return new URL(data.longUrl.startsWith('http') ? data.longUrl : `http://${data.longUrl}`).hostname
    } catch {
      return data.longUrl
    }
  })()

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-900 p-4">
      <div className="max-w-lg w-full bg-white rounded-2xl shadow-xl overflow-hidden">
        <div className="bg-blue-600 px-6 py-4">
          <h1 className="text-white text-lg font-semibold">Link Preview</h1>
          <p className="text-blue-200 text-sm">Verify where this link goes before visiting</p>
        </div>

        <div className="p-6 space-y-5">
          <div>
            <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">Short Link</label>
            <div className="mt-1 font-mono text-lg text-blue-600 font-semibold">
              {window.location.origin}/{data.code}
            </div>
          </div>

          {data.title && (
            <div>
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">Title</label>
              <div className="mt-1 text-gray-800 font-medium">{data.title}</div>
            </div>
          )}

          <div>
            <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">Destination</label>
            <div className="mt-1 text-gray-800 break-all">{data.longUrl}</div>
            <div className="mt-1 text-sm text-gray-500">
              Domain: <span className="font-medium">{destinationHost}</span>
            </div>
          </div>

          <div className="flex gap-6">
            <div>
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">Created</label>
              <div className="mt-1 text-gray-700">{createdDate}</div>
            </div>
            <div>
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">Total Clicks</label>
              <div className="mt-1 text-gray-700">{Number(data.totalClicks || 0).toLocaleString()}</div>
            </div>
            <div>
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">Status</label>
              <div className="mt-1">
                <span
                  className={`inline-block px-2 py-0.5 text-xs font-semibold rounded-full ${
                    data.isActive ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                  }`}
                >
                  {data.isActive ? 'Active' : 'Inactive'}
                </span>
              </div>
            </div>
          </div>

          {data.tags && data.tags.length > 0 && (
            <div>
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wide">Tags</label>
              <div className="mt-1 flex flex-wrap gap-1">
                {data.tags.map((tag) => (
                  <span key={tag} className="px-2 py-0.5 bg-gray-100 text-gray-600 text-xs rounded-full">
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          )}

          <div className="pt-2 flex gap-3">
            <a
              href={data.longUrl.startsWith('http') ? data.longUrl : `http://${data.longUrl}`}
              rel="noopener noreferrer"
              className="flex-1 text-center px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition font-medium"
            >
              Visit Link
            </a>
            <button
              onClick={() => navigate('/')}
              className="px-6 py-3 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition"
            >
              Go Back
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
