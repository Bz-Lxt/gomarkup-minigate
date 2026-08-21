const BASE = '/api/v1'

async function req(path, opts = {}) {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
    ...opts,
  })
  const text = await res.text()
  let body = null
  try {
    body = text ? JSON.parse(text) : null
  } catch {
    throw new Error('响应不是 JSON')
  }
  if (!res.ok || (body && body.code && body.code !== 0)) {
    const msg = (body && body.message) || res.statusText
    const err = new Error(msg)
    err.status = res.status
    throw err
  }
  return body ? body.data : null
}

export const api = {
  health: () => req('/health'),
  stats: () => req('/stats'),
  routes: () => req('/routes'),
  createRoute: (body) => req('/routes', { method: 'POST', body: JSON.stringify(body) }),
  updateRoute: (id, body) => req('/routes/' + encodeURIComponent(id), { method: 'PUT', body: JSON.stringify(body) }),
  toggleRoute: (id) => req('/routes/' + encodeURIComponent(id) + '/toggle', { method: 'PATCH' }),
  deleteRoute: (id) => req('/routes/' + encodeURIComponent(id), { method: 'DELETE' }),
  upstreams: () => req('/upstreams'),
  createUpstream: (body) => req('/upstreams', { method: 'POST', body: JSON.stringify(body) }),
  updateUpstream: (id, body) => req('/upstreams/' + encodeURIComponent(id), { method: 'PUT', body: JSON.stringify(body) }),
  deleteUpstream: (id) => req('/upstreams/' + encodeURIComponent(id), { method: 'DELETE' }),
  middlewares: () => req('/middlewares'),
  updateMW: (name, body) => req('/middlewares/' + encodeURIComponent(name), { method: 'PUT', body: JSON.stringify(body) }),
  config: () => req('/config'),
  configStatus: () => req('/config/status'),
  demoToken: () => req('/tokens/demo', { method: 'POST' }),
}
