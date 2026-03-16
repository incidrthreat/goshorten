import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { useAuth } from './hooks/useAuth'
import Layout from './components/Layout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import CreateURL from './pages/CreateURL'
import URLDetail from './pages/URLDetail'
import Tags from './pages/Tags'
import APIKeys from './pages/APIKeys'
import SettingsPage from './pages/SettingsPage'
import Preview from './pages/Preview'

export default function App() {
  const { user, loading, login, logout } = useAuth()
  const location = useLocation()

  // Public routes (no auth required)
  if (location.pathname.startsWith('/preview/')) {
    return (
      <Routes>
        <Route path="/preview/:code" element={<Preview />} />
      </Routes>
    )
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-gray-500">Loading...</div>
      </div>
    )
  }

  if (!user) {
    return <Login onLogin={login} />
  }

  return (
    <Layout user={user} onLogout={logout}>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/create" element={<CreateURL />} />
        <Route path="/urls/:code" element={<URLDetail />} />
        <Route path="/tags" element={<Tags />} />
        <Route path="/api-keys" element={<APIKeys />} />
        <Route path="/settings" element={<SettingsPage user={user} />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  )
}
