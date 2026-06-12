import { createContext, useContext, useEffect, useState } from "react"
import type { User } from "../lib/api"
import * as api from "../lib/api"

interface AuthState {
  user: User | null
  loading: boolean
  login: (email: string, password: string, remember?: boolean) => Promise<void>
  register: (email: string, password: string, name: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const cachedUser = api.getStoredUser()
  const [user, setUser] = useState<User | null>(cachedUser)
  const [loading, setLoading] = useState(!cachedUser)

  useEffect(() => {
    if (api.isAuthenticated()) {
      api.auth.me().then((u) => {
        setUser(u)
        api.setStoredUser(u)
      }).catch(() => {
        api.clearToken()
        setUser(null)
      }).finally(() => setLoading(false))
    } else {
      setLoading(false)
    }
  }, [])

  const login = async (email: string, password: string, remember = true) => {
    const res = await api.auth.login({ email, password })
    api.setToken(res.token, remember)
    api.setStoredUser(res.user)
    setUser(res.user)
  }

  const register = async (email: string, password: string, name: string) => {
    const res = await api.auth.register({ email, password, name })
    api.setToken(res.token)
    api.setStoredUser(res.user)
    setUser(res.user)
  }

  const logout = () => {
    api.clearToken()
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used within AuthProvider")
  return ctx
}
