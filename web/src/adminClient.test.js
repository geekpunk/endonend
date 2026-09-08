import { describe, it, expect, vi, afterEach } from 'vitest'
import { adminGraphqlRequest } from './adminClient'

describe('adminGraphqlRequest', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends the token in the X-Admin-Token header and returns data', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ data: { submitManifestUrl: { accepted: true } } }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const data = await adminGraphqlRequest('mutation {}', {}, 'secret-token')

    expect(data).toEqual({ submitManifestUrl: { accepted: true } })
    const [, options] = fetchMock.mock.calls[0]
    expect(options.headers['X-Admin-Token']).toBe('secret-token')
  })

  it('throws a clear message on 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 401 }))
    await expect(adminGraphqlRequest('mutation {}', {}, 'wrong')).rejects.toThrow(/unauthorized/i)
  })

  it('throws on other non-ok statuses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500, statusText: 'Internal Server Error' }))
    await expect(adminGraphqlRequest('mutation {}', {}, 'token')).rejects.toThrow(/500/)
  })

  it('throws when the GraphQL response carries errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ errors: [{ message: 'bad input' }] }) })
    )
    await expect(adminGraphqlRequest('mutation {}', {}, 'token')).rejects.toThrow(/bad input/)
  })
})
