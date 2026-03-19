const API_BASE = '/api/v1'

class APIError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers })

  if (!res.ok) {
    const body = await res.text()
    let message = `Request failed: ${res.status}`
    try {
      const json = JSON.parse(body)
      message = json.message || json.error || message
    } catch {
      if (body) message = body
    }
    throw new APIError(res.status, message)
  }

  return res.json()
}

// --- Auth ---
export const auth = {
  login: (email: string, password: string) =>
    request<{ token: string; user: { id: string; email: string; name: string; role: string } }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  me: () => request<{ id: string; email: string; name: string; role: string }>('/auth/me'),
  changePassword: (currentPassword: string, newPassword: string) =>
    request<{ status: string }>('/auth/change-password', {
      method: 'POST',
      body: JSON.stringify({ currentPassword, newPassword }),
    }),
  updateProfile: (params: { email?: string; name?: string }) =>
    request<{ id: string; email: string; name: string; role: string }>('/auth/profile', {
      method: 'PATCH',
      body: JSON.stringify(params),
    }),
}

// --- Admin ---
export const admin = {
  users: {
    list: (params: { search?: string; page?: number; page_size?: number }) => {
      const qs = new URLSearchParams()
      if (params.search) qs.set('search', params.search)
      if (params.page) qs.set('page', String(params.page))
      if (params.page_size) qs.set('page_size', String(params.page_size))
      return request<{ users: Array<Record<string, unknown>>; total: number; page: number; pageSize: number }>(
        `/admin/users?${qs}`
      )
    },
    update: (
      id: number,
      params: { role?: string; isActive?: boolean; email?: string; name?: string }
    ) =>
      request<Record<string, unknown>>(`/admin/users/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(params),
      }),
    create: (params: { email: string; name?: string; password: string; role?: string }) =>
      request<Record<string, unknown>>('/admin/users', {
        method: 'POST',
        body: JSON.stringify(params),
      }),
    delete: (id: number) =>
      request<{ status: string }>(`/admin/users/${id}`, { method: 'DELETE' }),
  },
  urls: {
    list: (params: Record<string, string | number>) => {
      const qs = new URLSearchParams()
      for (const [k, v] of Object.entries(params)) {
        if (v !== '' && v !== 0) qs.set(k, String(v))
      }
      return request<{
        urls: Array<Record<string, unknown>>
        total: number
        page: number
        pageSize: number
      }>(`/admin/short-urls?${qs}`)
    },
    get: (code: string) => request<Record<string, unknown>>(`/admin/short-urls/${code}`),
    assign: (code: string, assignedUserId: number) =>
      request<{ status: string }>(`/admin/short-urls/${code}`, {
        method: 'PATCH',
        body: JSON.stringify({ assignedUserId }),
      }),
  },
}

// --- URLs ---
export const urls = {
  list: (params: Record<string, string | number>) => {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(params)) {
      if (v !== '' && v !== 0) qs.set(k, String(v))
    }
    return request<{
      urls: Array<Record<string, unknown>>
      total: number
      page: number
      pageSize: number
    }>(`/short-urls?${qs}`)
  },
  create: (data: Record<string, unknown>) =>
    request<Record<string, unknown>>('/short-urls', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  get: (code: string) => request<Record<string, unknown>>(`/short-urls/${code}/resolve`),
  update: (code: string, data: Record<string, unknown>) =>
    request<Record<string, unknown>>(`/short-urls/${code}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),
  delete: (code: string) =>
    request<{ success: boolean }>(`/short-urls/${code}`, { method: 'DELETE' }),
  stats: (code: string) => request<Record<string, unknown>>(`/short-urls/${code}/stats`),
  qrCode: (code: string, size = 300) => `${API_BASE}/short-urls/${code}/qr-code?size=${size}`,
}

// --- Analytics ---
export const analytics = {
  summary: (code: string, excludeBots = false) =>
    request<Record<string, unknown>>(
      `/short-urls/${code}/visits/summary?excludeBots=${excludeBots}`
    ),
  byDate: (code: string, excludeBots = false) =>
    request<{ code: string; entries: Array<{ date: string; visits: string }> }>(
      `/short-urls/${code}/visits/by-date?excludeBots=${excludeBots}`
    ),
  byField: (code: string, field: string, excludeBots = false, limit = 10) =>
    request<{ code: string; field: string; entries: Array<{ value: string; visits: string }> }>(
      `/short-urls/${code}/visits/by-field?field=${field}&excludeBots=${excludeBots}&limit=${limit}`
    ),
  recent: (code: string, limit = 20, excludeBots = false) =>
    request<{ code: string; visits: Array<Record<string, unknown>> }>(
      `/short-urls/${code}/visits/recent?limit=${limit}&excludeBots=${excludeBots}`
    ),
  orphan: (limit = 20) =>
    request<{ visits: Array<Record<string, unknown>>; totalCount: string }>(
      `/visits/orphan?limit=${limit}`
    ),
}

// --- Tags ---
export const tags = {
  list: () => request<{ tags: Array<{ id: string; name: string; urlCount: string }> }>('/tags'),
  create: (name: string) =>
    request<{ id: string; name: string; urlCount: string }>('/tags', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  rename: (oldName: string, newName: string) =>
    request<{ id: string; name: string; urlCount: string }>(`/tags/${oldName}`, {
      method: 'PATCH',
      body: JSON.stringify({ newName }),
    }),
  delete: (name: string) =>
    request<{ success: boolean }>(`/tags/${name}`, { method: 'DELETE' }),
  stats: (name: string) =>
    request<{ tag: { id: string; name: string; urlCount: string }; totalClicks: string; uniqueUrls: string }>(
      `/tags/${name}/stats`
    ),
}

// --- API Keys ---
export const apiKeys = {
  list: () =>
    request<{ keys: Array<Record<string, unknown>> }>('/auth/api-keys'),
  create: (label: string, scopes: string) =>
    request<{ plaintextKey: string; key: Record<string, unknown> }>('/auth/api-keys', {
      method: 'POST',
      body: JSON.stringify({ label, scopes }),
    }),
  revoke: (keyId: number) =>
    request<{ success: boolean }>(`/auth/api-keys/${keyId}`, { method: 'DELETE' }),
  roll: (keyId: number) =>
    request<{ plaintextKey: string; key: Record<string, unknown> }>(`/auth/api-keys/${keyId}/roll`, {
      method: 'POST',
    }),
}

export { APIError }
