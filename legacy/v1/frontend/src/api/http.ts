import axios, { type AxiosInstance } from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiResponse } from '@/types'

const http: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

http.interceptors.request.use((config) => {
  const uid = localStorage.getItem('gallant-mock-uid')
  if (uid) config.headers['X-User-Id'] = uid
  return config
})

http.interceptors.response.use(
  (resp) => {
    // 文件下载等非 JSON 直接放行
    const ct = String(resp.headers['content-type'] || '')
    if (!ct.includes('application/json')) return resp

    const body = resp.data as ApiResponse<any>
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 0) return body.data as any
      ElMessage.error(body.message || `服务异常 (${body.code})`)
      return Promise.reject(new Error(body.message))
    }
    return resp.data
  },
  (err) => {
    const msg = err?.response?.data?.message || err.message || '网络请求失败'
    ElMessage.error(msg)
    return Promise.reject(err)
  }
)

export default http
