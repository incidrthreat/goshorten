import { Link, useLocation } from 'react-router-dom'
import {
  Link2,
  PlusCircle,
  Tags,
  Key,
  Settings,
  LogOut,
  BarChart3,
  Menu,
  X,
  Users,
  Moon,
  Sun,
} from 'lucide-react'
import { useState, useEffect } from 'react'

interface LayoutProps {
  children: React.ReactNode
  user: { email: string; role: string } | null
  onLogout: () => void
}

const baseNavItems = [
  { to: '/', icon: Link2, label: 'URLs' },
  { to: '/create', icon: PlusCircle, label: 'Create' },
  { to: '/tags', icon: Tags, label: 'Tags' },
  { to: '/api-keys', icon: Key, label: 'API Keys' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

const adminNavItems = [{ to: '/admin/users', icon: Users, label: 'Users' }]

function useDarkMode() {
  const [dark, setDark] = useState(() => {
    const stored = localStorage.getItem('darkMode')
    if (stored !== null) return stored === 'true'
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  })

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    localStorage.setItem('darkMode', String(dark))
  }, [dark])

  return [dark, setDark] as const
}

export default function Layout({ children, user, onLogout }: LayoutProps) {
  const location = useLocation()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [dark, setDark] = useDarkMode()

  const navItems = [
    ...baseNavItems,
    ...(user?.role === 'admin' ? adminNavItems : []),
  ]

  const allNav = [...baseNavItems, ...adminNavItems]

  return (
    <div className="min-h-screen flex dark:bg-gray-950">
      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-30 w-64 bg-gray-900 text-white transform transition-transform lg:translate-x-0 lg:static lg:flex lg:flex-col ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between h-16 px-6 border-b border-gray-700">
          <Link to="/" className="flex items-center gap-2 text-xl font-bold">
            <BarChart3 className="w-6 h-6 text-blue-400" />
            GoShorten
          </Link>
          <button className="lg:hidden" onClick={() => setSidebarOpen(false)}>
            <X className="w-5 h-5" />
          </button>
        </div>

        <nav className="flex-1 px-3 py-4 space-y-1">
          {navItems.map((item) => {
            const active = location.pathname === item.to
            return (
              <Link
                key={item.to}
                to={item.to}
                onClick={() => setSidebarOpen(false)}
                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  active
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                }`}
              >
                <item.icon className="w-5 h-5" />
                {item.label}
              </Link>
            )
          })}
        </nav>

        <div className="px-3 py-4 border-t border-gray-700">
          <div className="px-3 py-2 text-sm text-gray-400 truncate">
            {user?.email}
            {user?.role === 'admin' && (
              <span className="ml-2 px-1.5 py-0.5 text-xs bg-yellow-600 text-white rounded">
                admin
              </span>
            )}
          </div>
          <button
            onClick={onLogout}
            className="flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-sm font-medium text-gray-300 hover:bg-gray-800 hover:text-white transition-colors"
          >
            <LogOut className="w-5 h-5" />
            Sign out
          </button>
        </div>
      </aside>

      {/* Overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-20 bg-black/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Main */}
      <div className="flex-1 flex flex-col min-w-0">
        <header className="sticky top-0 z-10 h-16 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 flex items-center px-4 lg:px-8">
          <button className="lg:hidden mr-4" onClick={() => setSidebarOpen(true)}>
            <Menu className="w-6 h-6 dark:text-gray-300" />
          </button>
          <h1 className="text-lg font-semibold text-gray-800 dark:text-gray-100 truncate flex-1">
            {allNav.find((n) => n.to === location.pathname)?.label || 'GoShorten'}
          </h1>
          <button
            onClick={() => setDark(!dark)}
            className="p-2 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            title={dark ? 'Switch to light mode' : 'Switch to dark mode'}
          >
            {dark ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
          </button>
        </header>
        <main className="flex-1 p-4 lg:p-8 overflow-auto dark:bg-gray-950 dark:text-gray-100">
          {children}
        </main>
      </div>
    </div>
  )
}
