import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

let redirectingToLogin = false
let redirectingToSettings = false
export const TOKEN_KEY = 'zhiyu_waf_token'

const PRO_ERROR_MESSAGES = {
  professional_required: '此功能需要专业版授权，请在系统设置中激活。',
  feature_not_licensed: '当前授权不包含此功能，请检查授权范围。',
  license_unusable: '专业版授权当前不可用，请在系统设置中检查授权状态。',
}

export function getAuthToken() {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setAuthToken(token) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearAuthToken() {
  localStorage.removeItem(TOKEN_KEY)
}

api.interceptors.request.use(config => {
  const token = getAuthToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  res => res.data,
  err => {
    if (err.response?.status === 401) {
      clearAuthToken()
      if (!redirectingToLogin && window.location.pathname !== '/login') {
        redirectingToLogin = true
        window.location.replace('/login')
      }
      return Promise.reject(err)
    }

    const code = err.response?.data?.code
    if (err.response?.status === 403 && PRO_ERROR_MESSAGES[code]) {
      if (!err.config?.suppressError) {
        ElMessage.warning(PRO_ERROR_MESSAGES[code])
      }
      if (!redirectingToSettings && !window.location.pathname.startsWith('/settings')) {
        redirectingToSettings = true
        const reason = encodeURIComponent(code)
        window.setTimeout(() => {
          window.location.assign(`/settings?upgrade=${reason}`)
        }, 600)
      }
      return Promise.reject(err)
    }

    if (!err.config?.suppressError) {
      const msg = err.response?.data?.error || err.message || '请求失败'
      ElMessage.error(msg)
    }
    return Promise.reject(err)
  }
)

export default api
