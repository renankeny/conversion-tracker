import { getRequestHeader, getRequestURL, setResponseHeader, setResponseStatus } from 'h3'

const FORWARDED_HEADERS = [
  'user-agent',
  'accept-language',
  'referer',
  'cookie',
  'purpose',
  'sec-purpose',
  'x-purpose',
  'x-moz',
  'sec-ch-ua',
  'sec-ch-ua-mobile',
  'sec-ch-ua-platform',
  'cf-connecting-ip',
  'x-forwarded-for',
  'x-real-ip',
] as const

export default defineEventHandler(async (event) => {
  const slug = String(event.context.params?.slug || '').trim()
  if (!slug) {
    setResponseStatus(event, 404)
    return 'Not Found'
  }

  const config = useRuntimeConfig(event)
  const internalOrigin = String(config.internalApiUrl || '')
    .trim()
    .replace(/\/+$/, '')
    .replace(/\/api\/v1$/, '')
  const publicOrigin = String(config.public.apiUrl || '')
    .trim()
    .replace(/\/+$/, '')
    .replace(/\/api\/v1$/, '')
  const apiOrigin = internalOrigin || publicOrigin || 'http://127.0.0.1:8080'

  const reqUrl = getRequestURL(event)
  const upstreamUrl = `${apiOrigin}/w/${encodeURIComponent(slug)}${reqUrl.search || ''}`

  const headers = new Headers()
  for (const name of FORWARDED_HEADERS) {
    const val = getRequestHeader(event, name)
    if (val) headers.set(name, val)
  }

  const proto = getRequestHeader(event, 'x-forwarded-proto') || reqUrl.protocol.replace(/:$/, '') || 'https'
  const host = getRequestHeader(event, 'x-forwarded-host') || getRequestHeader(event, 'host') || reqUrl.host
  headers.set('x-forwarded-proto', proto)
  if (host) headers.set('x-forwarded-host', host)

  if (!headers.has('x-forwarded-for')) {
    const sockIp = event.node.req.socket?.remoteAddress
    if (sockIp) headers.set('x-forwarded-for', sockIp)
  }

  try {
    const upstream = await fetch(upstreamUrl, {
      method: 'GET',
      headers,
      redirect: 'manual',
      signal: AbortSignal.timeout(4000),
    })

    setResponseStatus(event, upstream.status)
    const contentType = upstream.headers.get('content-type') || 'text/html; charset=utf-8'
    setResponseHeader(event, 'content-type', contentType)

    const cacheControl = upstream.headers.get('cache-control') || 'no-store'
    setResponseHeader(event, 'cache-control', cacheControl)

    const location = upstream.headers.get('location')
    if (location) setResponseHeader(event, 'location', location)

    const setCookies = typeof upstream.headers.getSetCookie === 'function'
      ? upstream.headers.getSetCookie()
      : []
    if (setCookies.length > 0) {
      event.node.res.setHeader('set-cookie', setCookies)
    }

    return await upstream.text()
  } catch {
    setResponseStatus(event, 502)
    setResponseHeader(event, 'content-type', 'text/html; charset=utf-8')
    return '<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><title>Redirecionando</title></head><body>Serviço temporariamente indisponível.</body></html>'
  }
})
