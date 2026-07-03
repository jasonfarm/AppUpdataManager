import axios from 'axios'
import { useAuthStore } from '../stores/auth'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

api.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      const auth = useAuthStore()
      auth.logout()
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api

export interface LoginPayload {
  username: string
  password: string
}

export function login(payload: LoginPayload) {
  return api.post('/login', payload)
}

export function listSoftware() {
  return api.get('/software')
}

export function uploadSoftware(formData: FormData) {
  return api.post('/software', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export function deleteSoftware(id: number) {
  return api.delete(`/software/${id}`)
}

export function setLatestSoftware(id: number) {
  return api.post(`/software/${id}/latest`)
}

export function updateSoftwareName(id: number, name: string) {
  return api.put(`/software/${id}/name`, { name })
}

export function listClientVersions() {
  return api.get('/client-versions')
}

export function uploadClientVersion(formData: FormData) {
  return api.post('/client-versions', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export function setLatestClientVersion(id: number) {
  return api.post(`/client-versions/${id}/latest`)
}

export function listClients() {
  return api.get('/clients')
}

export function clientAction(id: number, action: string, version?: string) {
  return api.post(`/clients/${id}/${action}`, version ? { version } : {})
}
