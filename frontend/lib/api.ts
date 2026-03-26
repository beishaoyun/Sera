import axios from 'axios'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080'

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器 - 添加认证令牌
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('accessToken')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器 - 处理令牌过期
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true

      const refreshToken = localStorage.getItem('refreshToken')
      if (refreshToken) {
        try {
          const response = await axios.post(`${API_BASE_URL}/api/v1/auth/refresh`, {
            refresh_token: refreshToken,
          })

          const newAccessToken = response.data.access_token
          localStorage.setItem('accessToken', newAccessToken)

          originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
          return api(originalRequest)
        } catch (refreshError) {
          localStorage.removeItem('accessToken')
          localStorage.removeItem('refreshToken')
          window.location.href = '/login'
          return Promise.reject(refreshError)
        }
      }
    }

    return Promise.reject(error)
  }
)

// API 服务
export const authService = {
  register: (email: string, password: string, name: string) =>
    api.post('/api/v1/auth/register', { email, password, name }),

  login: (email: string, password: string) =>
    api.post('/api/v1/auth/login', { email, password }),

  refreshToken: (refreshToken: string) =>
    api.post('/api/v1/auth/refresh', { refresh_token: refreshToken }),

  getCurrentUser: () => api.get('/api/v1/user'),
}

export const serverService = {
  list: () => api.get('/api/v1/servers'),
  get: (id: string) => api.get(`/api/v1/servers/${id}`),
  create: (data: { name: string; host: string; port: number; username: string; password: string }) =>
    api.post('/api/v1/servers', data),
  update: (id: string, data: Partial<{ name: string; host: string; port: number; username: string }>) =>
    api.put(`/api/v1/servers/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/servers/${id}`),
  connect: (id: string) => api.post(`/api/v1/servers/${id}/connect`),
  getStatus: (id: string) => api.get(`/api/v1/servers/${id}/status`),
}

export const deploymentService = {
  list: (limit = 20, offset = 0) =>
    api.get(`/api/v1/deployments?limit=${limit}&offset=${offset}`),
  get: (id: string) => api.get(`/api/v1/deployments/${id}`),
  create: (data: { server_id: string; project_id: string; branch?: string }) =>
    api.post('/api/v1/deployments', data),
  cancel: (id: string) => api.post(`/api/v1/deployments/${id}/cancel`),
  getLogs: (id: string) => api.get(`/api/v1/deployments/${id}/logs`),
}

export const projectService = {
  list: () => api.get('/api/v1/projects'),
  get: (id: string) => api.get(`/api/v1/projects/${id}`),
  create: (data: { name: string; repo_url: string }) =>
    api.post('/api/v1/projects', data),
  delete: (id: string) => api.delete(`/api/v1/projects/${id}`),
}
