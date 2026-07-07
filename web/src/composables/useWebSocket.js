import { ref, onUnmounted } from 'vue'
import { getAuthToken } from '../api'

export function useWebSocket(url, maxMessages = 200, options = {}) {
  const messages = ref([])
  const isConnected = ref(false)
  let ws = null
  let reconnectTimer = null
  let reconnectAttempts = 0
  const maxReconnectDelay = 30000
  const maxReconnectAttempts = options.maxReconnectAttempts ?? 8
  let disposed = false

  function getReconnectDelay() {
    const base = Math.min(1000 * Math.pow(2, reconnectAttempts), maxReconnectDelay)
    return base + Math.random() * 1000
  }

  function scheduleReconnect() {
    if (disposed || reconnectAttempts >= maxReconnectAttempts) return
    reconnectAttempts++
    reconnectTimer = setTimeout(connect, getReconnectDelay())
  }

  function connect() {
    if (disposed || ws?.readyState === WebSocket.OPEN || ws?.readyState === WebSocket.CONNECTING) return
    if (document.visibilityState === 'hidden') {
      scheduleReconnect()
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'

    // Use same host:port as the page (Vite proxy handles /api)
    const host = window.location.host
    const token = getAuthToken()
    if (!token) return
    const separator = url.includes('?') ? '&' : '?'
    ws = new WebSocket(`${protocol}//${host}${url}${separator}token=${encodeURIComponent(token)}`)

    ws.onopen = () => {
      isConnected.value = true
      reconnectAttempts = 0
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        messages.value.unshift(data)
        if (messages.value.length > maxMessages) {
          messages.value = messages.value.slice(0, maxMessages)
        }
      } catch {
        // ignore malformed messages
      }
    }

    ws.onclose = () => {
      isConnected.value = false
      ws = null
      scheduleReconnect()
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  connect()

  function handleVisibilityChange() {
    if (document.visibilityState === 'visible' && !isConnected.value) {
      reconnectAttempts = Math.min(reconnectAttempts, 2)
      if (reconnectTimer) clearTimeout(reconnectTimer)
      reconnectTimer = setTimeout(connect, 300)
    }
  }

  document.addEventListener('visibilitychange', handleVisibilityChange)

  onUnmounted(() => {
    disposed = true
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    if (reconnectTimer) clearTimeout(reconnectTimer)
    if (ws) ws.close(1000, 'component unmounted')
  })

  return { messages, isConnected }
}
