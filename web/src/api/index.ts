import axios from 'axios'
import type { AxiosInstance, AxiosResponse } from 'axios'

const instance: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Request interceptor - add JWT token
instance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor - handle 401 unauthorized
instance.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Lazy import to avoid circular dependency (auth store imports types from this module)
      Promise.all([
        import('@/stores/auth'),
        import('@/router')
      ]).then(([{ useAuthStore }, { default: router }]) => {
        useAuthStore().clearAuth()
        router.push('/login')
      })
    }
    return Promise.reject(error)
  }
)

// Helper: extract error message from axios errors
export function getErrorMessage(error: unknown, fallback: string): string {
  if (axios.isAxiosError(error) && typeof error.response?.data?.error === 'string') {
    return error.response.data.error
  }
  return fallback
}

// Types
export interface User {
  id: number
  username: string
  is_admin: boolean
}

export interface LoginResponse {
  token: string
  user: User
}

export interface Account {
  id: number
  name: string
  platform: 'qq' | 'wx'
  code: string
  auto_start: boolean
  farm_interval: number
  friend_interval: number
  enable_steal: boolean
  force_lowest: boolean
  status: 'running' | 'stopped' | 'error'
  level: number
  gold: number
  exp: number
  created_at: string
  updated_at: string
}

export interface CreateAccountRequest {
  name: string
  platform: 'qq' | 'wx'
  code: string
  auto_start: boolean
  farm_interval: number
  friend_interval: number
  enable_steal: boolean
  force_lowest: boolean
}

export interface BotStatus {
  running: boolean
  level: number
  gold: number
  exp: number
  current_exp: number
  max_exp: number
  nickname: string
  platform: string
  started_at: string | null
}

export interface LandStatus {
  id: number
  level: number
  max_level: number
  unlocked: boolean
  crop_name?: string
  crop_id?: number
  phase?: string
}

export interface DashboardStats {
  total_accounts: number
  running_bots: number
  total_gold: number
  accounts: Array<{
    id: number
    name: string
    level: number
    gold: number
    exp: number
    status: string
    platform: string
    total_steal: number
    total_help: number
    friends_count: number
    total_lands: number
    unlocked_lands: number
    lands: LandStatus[]
    // Level up estimation
    exp_rate_per_hour: number
    next_level_exp: number
    exp_to_next_level: number
    hours_to_next_level: number
  }>
}

export interface LogEntry {
  id: number
  account_id: number
  created_at: string
  level: 'info' | 'warn' | 'error' | 'debug'
  tag: string
  message: string
}

export interface QRCodeResponse {
  login_code: string
  qr_code_url: string
  expires_at: string
}

export interface QRCodePollResponse {
  status: 'wait' | 'ok' | 'expired' | 'error'
  code?: string
  message?: string
}

export const authApi = {
  login: (username: string, password: string): Promise<AxiosResponse<LoginResponse>> => 
    instance.post('/auth/login', { username, password }),
  
  register: (username: string, password: string): Promise<AxiosResponse<LoginResponse>> => 
    instance.post('/auth/register', { username, password }),
  
  logout: (): Promise<AxiosResponse<void>> => 
    instance.post('/auth/logout')
}

export const accountApi = {
  getAll: (): Promise<AxiosResponse<Account[]>> => 
    instance.get('/accounts'),
  
  create: (data: CreateAccountRequest): Promise<AxiosResponse<Account>> => 
    instance.post('/accounts', data),
  
  update: (id: number, data: Partial<CreateAccountRequest>): Promise<AxiosResponse<Account>> => 
    instance.put(`/accounts/${id}`, data),
  
  delete: (id: number): Promise<AxiosResponse<void>> => 
    instance.delete(`/accounts/${id}`),
  
  start: (id: number): Promise<AxiosResponse<{ message: string }>> => 
    instance.post(`/accounts/${id}/start`),
  
  stop: (id: number): Promise<AxiosResponse<{ message: string }>> => 
    instance.post(`/accounts/${id}/stop`),
  
  getStatus: (id: number): Promise<AxiosResponse<BotStatus>> => 
    instance.get(`/accounts/${id}/status`),
  
  getQRCode: (id: number): Promise<AxiosResponse<QRCodeResponse>> => 
    instance.post(`/accounts/${id}/qrcode`),
  
  pollQRCode: (id: number, loginCode: string): Promise<AxiosResponse<QRCodePollResponse>> => 
    instance.get(`/accounts/${id}/qrcode/poll`, { params: { login_code: loginCode } }),
  
  getLogs: (id: number, limit: number = 100): Promise<AxiosResponse<LogEntry[]>> => 
    instance.get(`/accounts/${id}/logs`, { params: { limit } })
}

export const dashboardApi = {
  getStats: (): Promise<AxiosResponse<DashboardStats>> => 
    instance.get('/dashboard')
}

export const logsApi = {
  getHistorical: (accountId: number, limit: number = 100): Promise<AxiosResponse<LogEntry[]>> => 
    instance.get(`/accounts/${accountId}/logs`, { params: { limit } })
}

// WebSocket connection for real-time logs
export function createLogWebSocket(accountId: number): WebSocket {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const token = localStorage.getItem('token')
  return new WebSocket(`${protocol}//${host}/api/ws/logs?account_id=${accountId}&token=${token}`)
}

export default instance
