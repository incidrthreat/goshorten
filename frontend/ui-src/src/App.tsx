import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { useEffect } from 'react'
import { useAuth } from './hooks/useAuth'
import Layout from './components/Layout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import CreateURL from './pages/CreateURL'
import EditURL from './pages/EditURL'
import URLDetail from './pages/URLDetail'
import Tags from './pages/Tags'
import APIKeys from './pages/APIKeys'
import SettingsPage from './pages/SettingsPage'
import Preview from './pages/Preview'
import Expired from './pages/Expired'
import AdminUsers from './pages/admin/Users'
import AdminOIDCProviders from './pages/admin/OIDCProviders'
import OIDCCallback from './pages/OIDCCallback'

export default function App() {
  const { user, loading, login, logout, checkAuth, theme, setTheme, isOIDC } = useAuth()
  const location = useLocation()

  // Apply theme class to document
  useEffect(() => {
    if (theme === 'dark') {
      document.documentElement.classList.add('dark')
    } else if (theme === 'light') {
      document.documentElement.classList.remove('dark')
    } else {
      // system
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      document.documentElement.classList.toggle('dark', prefersDark)
    }
  }, [theme])

  // Public routes (no auth required)
  if (location.pathname.startsWith('/preview/')) {
    return (
      <Routes>
        <Route path="/preview/:code" element={<Preview />} />
      </Routes>
    )
  }

  if (location.pathname === '/expired') {
    return (
      <Routes>
        <Route path="/expired" element={<Expired />} />
      </Routes>
    )
  }

  if (location.pathname === '/auth/callback') {
    return (
      <Routes>
        <Route path="/auth/callback" element={<OIDCCallback />} />
      </Routes>
    )
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-950">
        <div className="text-gray-500">Loading...</div>
      </div>
    )
  }

  if (!user) {
    return <Login onLogin={login} />
  }

  return (
    <Layout user={user} onLogout={logout} theme={theme} onSetTheme={setTheme}>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/create" element={<CreateURL />} />
        <Route path="/urls/:code" element={<URLDetail />} />
        <Route path="/urls/:code/edit" element={<EditURL />} />
        <Route path="/tags" element={<Tags />} />
        <Route path="/api-keys" element={<APIKeys />} />
        <Route
          path="/settings"
          element={
            <SettingsPage
              user={user}
              onRefreshUser={checkAuth}
              isOIDC={isOIDC}
              theme={theme}
              onSetTheme={setTheme}
            />
          }
        />
        {user.role === 'admin' && (
          <>
            <Route path="/admin/users" element={<AdminUsers />} />
            <Route path="/admin/oidc-providers" element={<AdminOIDCProviders />} />
          </>
        )}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  )
}
