import { useState, useEffect, useCallback } from 'react'
import { auth } from '../api/client'

interface User {
  id: string
  email: string
  name: string
  role: string
}

export function useAuth() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [isOIDC, setIsOIDC] = useState(false)
  const [theme, setThemeState] = useState<string>('system')

  const checkAuth = useCallback(async () => {
    const token = localStorage.getItem('token')
    if (!token) {
      setUser(null)
      setLoading(false)
      return
    }
    try {
      const [u, acct] = await Promise.all([auth.me(), auth.account()])
      setUser(u)
      setIsOIDC(acct.isOIDC)
      setThemeState(acct.theme || 'system')
    } catch {
      localStorage.removeItem('token')
      setUser(null)
      setIsOIDC(false)
      setThemeState('system')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    checkAuth()
  }, [checkAuth])

  const login = async (email: string, password: string) => {
    const res = await auth.login(email, password)
    localStorage.setItem('token', res.token)
    setUser(res.user)
    // Fetch account details after login
    try {
      const acct = await auth.account()
      setIsOIDC(acct.isOIDC)
      setThemeState(acct.theme || 'system')
    } catch {
      // non-fatal
    }
  }

  const logout = () => {
    localStorage.removeItem('token')
    setUser(null)
    setIsOIDC(false)
    setThemeState('system')
  }

  const setTheme = async (newTheme: string) => {
    setThemeState(newTheme)
    try {
      await auth.updatePreferences({ theme: newTheme })
    } catch {
      // non-fatal
    }
  }

  return { user, loading, login, logout, checkAuth, isOIDC, theme, setTheme }
}
