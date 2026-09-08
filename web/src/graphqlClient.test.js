import { describe, it, expect, vi, afterEach } from 'vitest'
import { graphqlRequest } from './graphqlClient'

describe('graphqlRequest', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('posts the query and returns data on success', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { artists: [{ name: 'Ligatures' }] } }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const data = await graphqlRequest('{ artists { name } }')

    expect(data).toEqual({ artists: [{ name: 'Ligatures' }] })
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/v1/graphql'),
      expect.objectContaining({ method: 'POST' })
    )
  })

  it('throws when the HTTP response is not ok', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 500, statusText: 'Internal Server Error' })
    )

    await expect(graphqlRequest('{ artists { name } }')).rejects.toThrow(/500/)
  })

  it('throws when the GraphQL response carries errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ errors: [{ message: 'boom' }] }),
      })
    )

    await expect(graphqlRequest('{ artists { name } }')).rejects.toThrow(/boom/)
  })
})
