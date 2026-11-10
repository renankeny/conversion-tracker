export interface CodeSnippetTab {
  name: string
  icon: string
  code: string
}

export function buildApiDocSnippets(apiUrl: string) {
  const sendEventTabs: CodeSnippetTab[] = [
    {
      name: 'cURL',
      icon: '💻',
      code: `curl -X POST ${apiUrl}/send-event \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" \\
  -d '{
    "pixel_name": "meu-pixel",
    "event_name": "Lead",
    "external_id": "user-123",
    "email": "cliente@example.com",
    "phone": "+5511999998888",
    "first_name": "João",
    "last_name": "Silva"
  }'`,
    },
    {
      name: 'PHP',
      icon: '🐘',
      code: `<?php
$apiKey = 'trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx';
$endpoint = '${apiUrl}/send-event';

$event = [
    'pixel_name' => 'meu-pixel',
    'event_name' => 'Lead',
    'external_id' => 'user-' . $userId,
    'email' => $userEmail,
    'phone' => $userPhone,
    'first_name' => $firstName,
    'last_name' => $lastName
];

$ch = curl_init($endpoint);
curl_setopt($ch, CURLOPT_POST, 1);
curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($event));
curl_setopt($ch, CURLOPT_HTTPHEADER, [
    'Content-Type: application/json',
    'X-API-Key: ' . $apiKey
]);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);

$response = curl_exec($ch);
curl_close($ch);
?>`,
    },
    {
      name: 'Node.js',
      icon: '🟢',
      code: `const axios = require('axios');

const apiKey = 'trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx';
const endpoint = '${apiUrl}/send-event';

async function sendLead(userData) {
  const response = await axios.post(endpoint, {
    pixel_name: 'meu-pixel',
    event_name: 'Lead',
    external_id: \`user-\${userData.id}\`,
    email: userData.email,
    phone: userData.phone,
    first_name: userData.firstName,
    last_name: userData.lastName
  }, {
    headers: { 'Content-Type': 'application/json', 'X-API-Key': apiKey }
  });
  console.log('Lead enviado:', response.data.event_id);
}`,
    },
    {
      name: 'Python',
      icon: '🐍',
      code: `import requests

api_key = 'trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
endpoint = '${apiUrl}/send-event'

event = {
    'pixel_name': 'meu-pixel',
    'event_name': 'Lead',
    'external_id': f'user-{user_id}',
    'email': user_email,
    'phone': user_phone,
    'first_name': first_name,
    'last_name': last_name
}

response = requests.post(endpoint, json=event, headers={'X-API-Key': api_key})`,
    },
  ]

  const crmLeadTabs: CodeSnippetTab[] = [
    {
      name: 'cURL',
      icon: '💻',
      code: `curl -X POST ${apiUrl}/admin/ext-api/crm/leads \\
  -H "Authorization: Bearer trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "first_name": "João",
    "last_name": "Silva",
    "email": "joao@exemplo.com",
    "phone": "5511999998888",
    "external_id": "user_123",
    "stage_id": 1,
    "value": 150.00
  }'`,
    },
    {
      name: 'Node.js',
      icon: '🟢',
      code: `const axios = require('axios');

async function createLead() {
  await axios.post('${apiUrl}/admin/ext-api/crm/leads', {
    first_name: 'João',
    last_name: 'Silva',
    email: 'joao@exemplo.com',
    phone: '5511999998888',
    external_id: 'user_123',
    stage_id: 1,
    value: 150.00
  }, {
    headers: { 'Authorization': 'Bearer trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx' }
  });
}`,
    },
  ]

  const lookupTabs: CodeSnippetTab[] = [
    {
      name: 'Node.js',
      icon: '🟢',
      code: `const axios = require('axios');

async function buscarUsuario(email) {
  const { data } = await axios.get('${apiUrl}/admin/tracked-users/lookup', {
    params: { email },
    headers: { 'Authorization': 'Bearer trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx' }
  });
  return data;
}`,
    },
    {
      name: 'Python',
      icon: '🐍',
      code: `import requests

r = requests.get(
    '${apiUrl}/admin/tracked-users/lookup',
    params={'phone': phone},
    headers={'Authorization': 'Bearer trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'}
)`,
    },
  ]

  const batchTabs: CodeSnippetTab[] = [
    {
      name: 'cURL',
      icon: '💻',
      code: `curl -X POST ${apiUrl}/send-events-batch \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" \\
  -d '{
    "events": [
      { "pixel_name": "meu-pixel", "event_name": "PageView", "external_id": "user-123" },
      { "pixel_name": "meu-pixel", "event_name": "Purchase", "external_id": "user-123", "value": 99.90, "currency": "BRL" }
    ]
  }'`,
    },
  ]

  const stageTabs: CodeSnippetTab[] = [
    {
      name: 'cURL',
      icon: '💻',
      code: `curl ${apiUrl}/admin/ext-api/crm/stages \\
  -H "Authorization: Bearer trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"`,
    },
  ]

  return { sendEventTabs, crmLeadTabs, lookupTabs, batchTabs, stageTabs }
}

