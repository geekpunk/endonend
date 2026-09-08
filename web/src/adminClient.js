// POST helper for the admin-only GraphQL API at /v1/admin/graphql, per
// KB/0010-mvp-scope.md: everything graphqlClient.js does, plus the
// X-Admin-Token header the backend's requireAdminSecret middleware checks.
const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export async function adminGraphqlRequest(query, variables, token) {
  const res = await fetch(`${API_URL}/v1/admin/graphql`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Admin-Token': token || '',
    },
    body: JSON.stringify({ query, variables }),
  })
  if (res.status === 401) {
    throw new Error('Unauthorized: check the admin token.')
  }
  if (!res.ok) {
    throw new Error(`Admin request failed: ${res.status} ${res.statusText}`)
  }
  const body = await res.json()
  if (body.errors && body.errors.length > 0) {
    throw new Error(body.errors.map((e) => e.message).join('; '))
  }
  return body.data
}

export const ADMIN_TOKEN_STORAGE_KEY = 'endonend_admin_token'
