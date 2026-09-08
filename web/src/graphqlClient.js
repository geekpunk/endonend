// Minimal GraphQL client for the versioned public API described in
// KB/0010-mvp-scope.md ("API (GraphQL)"). No codegen, no library: this is
// one small POST helper, matching the "minimal, no polish" MVP scope.

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export async function graphqlRequest(query, variables) {
  const res = await fetch(`${API_URL}/v1/graphql`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  if (!res.ok) {
    throw new Error(`GraphQL request failed: ${res.status} ${res.statusText}`)
  }
  const body = await res.json()
  if (body.errors && body.errors.length > 0) {
    throw new Error(body.errors.map((e) => e.message).join('; '))
  }
  return body.data
}
