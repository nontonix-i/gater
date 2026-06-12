const API_BASE = "/api/v1"

function getToken(): string | null {
  return localStorage.getItem("gater_token") || sessionStorage.getItem("gater_token")
}

export function setToken(token: string, remember = true) {
  if (remember) {
    localStorage.setItem("gater_token", token)
    sessionStorage.removeItem("gater_token")
  } else {
    sessionStorage.setItem("gater_token", token)
    localStorage.removeItem("gater_token")
  }
}

export function clearToken() {
  localStorage.removeItem("gater_token")
  sessionStorage.removeItem("gater_token")
  clearStoredUser()
}

const USER_KEY = "gater_user"

export function setStoredUser(user: User) {
  const data = JSON.stringify(user)
  if (localStorage.getItem("gater_token")) {
    localStorage.setItem(USER_KEY, data)
  } else {
    sessionStorage.setItem(USER_KEY, data)
  }
}

export function getStoredUser(): User | null {
  const raw = localStorage.getItem(USER_KEY) || sessionStorage.getItem(USER_KEY)
  if (!raw) return null
  try { return JSON.parse(raw) } catch { return null }
}

export function clearStoredUser() {
  localStorage.removeItem(USER_KEY)
  sessionStorage.removeItem(USER_KEY)
}

export function isAuthenticated(): boolean {
  return !!localStorage.getItem("gater_token") || !!sessionStorage.getItem("gater_token")
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  }
  if (token) {
    headers["X-API-Key"] = token
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })

  if (res.status === 401) {
    clearToken()
    window.location.href = "/login"
    throw new Error("Unauthorized")
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `Request failed: ${res.status}`)
  }

  return res.json()
}

export interface User {
  id: string
  email: string
  name: string
  api_key: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface RegisterRequest {
  email: string
  password: string
  name: string
}

export interface TaskResult {
  id: string
  task_id: string
  provider: string
  status: string
  source_url: string
  output_url: string
  file_code: string
  provider_file_name: string
  provider_file_size: number
  progress: number
  error_message: string
  error: string
  created_at: string
  started_at: string | null
  completed_at: string | null
}

export interface Task {
  id: string
  user_id: string
  status: string
  source_type: string
  source_url: string
  title: string
  file_name: string
  file_size: number
  file_path: string
  created_at: string
  completed_at: string | null
  results: TaskResult[]
}

export interface Provider {
  name: string
  type: string
  supports_anonymous: boolean
  supports_remote_url: boolean
  has_api: boolean
}

export interface UploadURLRequest {
  url: string
  providers: string[]
  title?: string
}

export interface UploadURLResponse {
  task_id: string
  status: string
}

// Auth
export const auth = {
  login: (data: LoginRequest) =>
    request<LoginResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
      headers: { "Content-Type": "application/json" },
    }),
  register: (data: RegisterRequest) =>
    request<LoginResponse>("/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
      headers: { "Content-Type": "application/json" },
    }),
  me: () => request<User>("/auth/me"),
}

// Tasks
export const tasks = {
  list: (params?: { limit?: number; offset?: number }) => {
    const query = new URLSearchParams()
    if (params?.limit) query.set("limit", String(params.limit))
    if (params?.offset) query.set("offset", String(params.offset))
    const qs = query.toString()
    return request<{ tasks: Task[]; total: number }>(
      `/tasks${qs ? `?${qs}` : ""}`
    )
  },
  get: (id: string) => request<Task>(`/task/${id}`),
  create: (data: UploadURLRequest) =>
    request<UploadURLResponse>("/upload/url", {
      method: "POST",
      body: JSON.stringify(data),
      headers: { "Content-Type": "application/json" },
    }),
}

// Providers
export const providers = {
  list: async () => {
    const res = await request<{ providers: Provider[] }>("/providers")
    return res.providers
  },
  get: (name: string) => request<Provider>(`/providers/${name}`),
  getCredentials: (name: string) =>
    request<{
      provider: string
      has_creds: boolean
      fields: { key: string; label: string; has_value: boolean }[]
    }>(`/providers/${name}/credentials`),
  saveCredentials: (name: string, values: Record<string, string>) =>
    request<{ status: string }>(`/providers/${name}/credentials`, {
      method: "PUT",
      body: JSON.stringify({ values }),
      headers: { "Content-Type": "application/json" },
    }),
}

// Settings
export interface Settings {
  default_providers: string[]
  api_key: string
}

export const settings = {
  get: () => request<Settings>("/settings"),
  update: (data: { default_providers: string[] }) =>
    request<{ status: string }>("/settings", {
      method: "PUT",
      body: JSON.stringify(data),
      headers: { "Content-Type": "application/json" },
    }),
  regenerateKey: () =>
    request<{ api_key: string }>("/auth/regenerate-key", {
      method: "POST",
    }),
}
