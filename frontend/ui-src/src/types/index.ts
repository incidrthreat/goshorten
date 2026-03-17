export interface ShortURL {
  code: string
  longUrl: string
  title: string
  createdAt: string
  expiresAt?: string
  isActive: boolean
  maxVisits: number
  redirectType: number
  isCrawlable: boolean
  domain: string
  tags: string[]
  totalClicks: number
}

export interface ListURLsResponse {
  urls: ShortURL[]
  total: number
  page: number
  pageSize: number
}

export interface StatsResponse {
  code: string
  longUrl: string
  title: string
  createdAt: string
  expiresAt?: string
  lastAccessed?: string
  totalClicks: number
  isActive: boolean
  maxVisits: number
  tags: string[]
}

export interface VisitSummary {
  code: string
  totalVisits: number
  uniqueVisitors: number
  botVisits: number
  humanVisits: number
}

export interface VisitsByDateEntry {
  date: string
  visits: number
}

export interface VisitsByFieldEntry {
  value: string
  visits: number
}

export interface VisitEntry {
  visitedAt: string
  ipAddress: string
  userAgent: string
  referer: string
  country: string
  city: string
  deviceType: string
  browser: string
  os: string
  isBot: boolean
}

export interface TagInfo {
  id: number
  name: string
  urlCount: number
}

export interface TagStats {
  tag: TagInfo
  totalClicks: number
  uniqueUrls: number
}

export interface APIKeyInfo {
  id: number
  label: string
  scopes: string
  createdAt: string
  expiresAt?: string
  revoked: boolean
  keyPrefix: string
}

export interface UserInfo {
  id: number
  email: string
  name: string
  role: string
  createdAt: string
}

export interface LoginResponse {
  token: string
  user: UserInfo
}
