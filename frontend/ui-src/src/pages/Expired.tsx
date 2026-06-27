import { useSearchParams } from 'react-router-dom'
import { LinkIcon } from 'lucide-react'

export default function Expired() {
  const [params] = useSearchParams()
  const code = params.get('code')

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-950">
      <div className="max-w-md w-full mx-auto p-8 text-center space-y-4">
        <div className="flex justify-center">
          <div className="w-16 h-16 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
            <LinkIcon className="w-8 h-8 text-gray-400 dark:text-gray-500" />
          </div>
        </div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Link Unavailable
        </h1>
        <p className="text-gray-500 dark:text-gray-400">
          {code ? (
            <>
              The link{' '}
              <code className="font-mono text-blue-600 dark:text-blue-400">/{code}</code>{' '}
              has expired or reached its maximum number of visits.
            </>
          ) : (
            'This link has expired or reached its maximum number of visits.'
          )}
        </p>
        <a
          href="/"
          className="inline-block px-5 py-2.5 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors"
        >
          Go to GoShorten
        </a>
      </div>
    </div>
  )
}
