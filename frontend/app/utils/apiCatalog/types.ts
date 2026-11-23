export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

export interface ApiParamItem {
  name: string
  location: 'path' | 'query' | 'body'
  type: string
  required?: boolean
  description: string
}

export interface ApiEndpointSpec {
  id: string
  method: HttpMethod
  path: string
  title: string
  summary: string
  note?: string
  params?: ApiParamItem[]
  sampleQuery?: string
  sampleBody?: Record<string, unknown>
  sampleResponse: string
}

export interface ApiModuleGroup {
  id: string
  badge: string
  title: string
  subtitle: string
  endpoints: ApiEndpointSpec[]
}

export interface CodeTabItem {
  name: string
  code: string
}

export function buildEndpointCodeTabs(apiUrl: string, ep: ApiEndpointSpec): CodeTabItem[] {
  const cleanBase = apiUrl.replace(/\/+$/, '')
  const resolvedPath = ep.path.replace(/\{([^}]+)\}/g, (_, key: string) => {
    if (key.toLowerCase().includes('event')) return 'EVT_12345'
    return '1'
  })
  const fullUrl = `${cleanBase}${resolvedPath}${ep.sampleQuery ? `?${ep.sampleQuery}` : ''}`
  const hasBody = Boolean(ep.sampleBody && (ep.method === 'POST' || ep.method === 'PUT'))
  const bodyJson = hasBody ? JSON.stringify(ep.sampleBody, null, 2) : ''

  const curlLines = [
    `curl -X ${ep.method} "${fullUrl}" \\`,
    `  -H "X-API-Key: trk_sua_chave_aqui"`,
  ]
  if (hasBody) {
    curlLines[1] += ' \\'
    curlLines.push(`  -H "Content-Type: application/json" \\`)
    curlLines.push(`  -d '${bodyJson}'`)
  }

  const nodeLines = hasBody
    ? [
        `const response = await fetch("${fullUrl}", {`,
        `  method: "${ep.method}",`,
        `  headers: {`,
        `    "X-API-Key": "trk_sua_chave_aqui",`,
        `    "Content-Type": "application/json",`,
        `  },`,
        `  body: JSON.stringify(${bodyJson.replace(/\n/g, '\n  ')}),`,
        `});`,
        `const data = await response.json();`,
      ]
    : [
        `const response = await fetch("${fullUrl}", {`,
        `  method: "${ep.method}",`,
        `  headers: { "X-API-Key": "trk_sua_chave_aqui" },`,
        `});`,
        `const data = await response.json();`,
      ]

  const pyLines = hasBody
    ? [
        `import requests`,
        ``,
        `payload = ${bodyJson}`,
        `res = requests.request(`,
        `    "${ep.method}",`,
        `    "${fullUrl}",`,
        `    headers={"X-API-Key": "trk_sua_chave_aqui"},`,
        `    json=payload,`,
        `)`,
        `print(res.json())`,
      ]
    : [
        `import requests`,
        ``,
        `res = requests.request(`,
        `    "${ep.method}",`,
        `    "${fullUrl}",`,
        `    headers={"X-API-Key": "trk_sua_chave_aqui"},`,
        `)`,
        `print(res.json())`,
      ]

  const phpLines = [
    `<?php`,
    `$ch = curl_init("${fullUrl}");`,
    `curl_setopt($ch, CURLOPT_CUSTOMREQUEST, "${ep.method}");`,
    `curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);`,
    `curl_setopt($ch, CURLOPT_HTTPHEADER, [`,
    `    "X-API-Key: trk_sua_chave_aqui",`,
    ...(hasBody ? [`    "Content-Type: application/json",`] : []),
    `]);`,
    ...(hasBody ? [`curl_setopt($ch, CURLOPT_POSTFIELDS, '${JSON.stringify(ep.sampleBody)}');`] : []),
    `$response = curl_exec($ch);`,
    `curl_close($ch);`,
  ]

  return [
    { name: 'cURL', code: curlLines.join('\n') },
    { name: 'Node.js', code: nodeLines.join('\n') },
    { name: 'Python', code: pyLines.join('\n') },
    { name: 'PHP', code: phpLines.join('\n') },
  ]
}

export function buildMcpSetupTabs(apiUrl: string): CodeTabItem[] {
  const mcpUrl = `${apiUrl.replace(/\/+$/, '')}/mcp`

  return [
    {
      name: 'n8n (MCP Client)',
      code: [
        `Endpoint URL: ${mcpUrl}`,
        `Transport: HTTP Streamable`,
        `Authentication: Header Auth`,
        `Header Name: X-API-Key`,
        `Header Value: trk_sua_chave_aqui`,
      ].join('\n'),
    },
    {
      name: 'Claude Desktop',
      code: JSON.stringify(
        {
          mcpServers: {
            trackeado: {
              command: 'npx',
              args: [
                '-y',
                'mcp-remote',
                mcpUrl,
                '--header',
                'X-API-Key: trk_sua_chave_aqui',
              ],
            },
          },
        },
        null,
        2
      ),
    },
    {
      name: 'Cursor / Antigravity',
      code: JSON.stringify(
        {
          mcpServers: {
            trackeado: {
              url: mcpUrl,
              headers: {
                'X-API-Key': 'trk_sua_chave_aqui',
              },
            },
          },
        },
        null,
        2
      ),
    },
    {
      name: 'Teste cURL (MCP)',
      code: [
        `curl -X POST "${mcpUrl}" \\`,
        `  -H "X-API-Key: trk_sua_chave_aqui" \\`,
        `  -H "Content-Type: application/json" \\`,
        `  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'`,
      ].join('\n'),
    },
  ]
}
