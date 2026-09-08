import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import AdminConsole from './AdminConsole'
import * as adminClient from './adminClient'

beforeEach(() => {
  sessionStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('AdminConsole', () => {
  it('persists the token to sessionStorage as it is typed', () => {
    render(<AdminConsole />)
    fireEvent.change(screen.getByLabelText(/admin token/i), { target: { value: 'my-secret' } })
    expect(sessionStorage.getItem(adminClient.ADMIN_TOKEN_STORAGE_KEY)).toBe('my-secret')
  })

  it('restores a previously saved token on mount', () => {
    sessionStorage.setItem(adminClient.ADMIN_TOKEN_STORAGE_KEY, 'saved-token')
    render(<AdminConsole />)
    expect(screen.getByLabelText(/admin token/i)).toHaveValue('saved-token')
  })

  it('submits a URL and shows the accepted result', async () => {
    const spy = vi.spyOn(adminClient, 'adminGraphqlRequest').mockResolvedValue({
      submitManifestUrl: { accepted: true, errors: [], identity: { name: 'Ligatures' } },
    })
    render(<AdminConsole />)

    fireEvent.change(screen.getByLabelText(/admin token/i), { target: { value: 'tok' } })
    fireEvent.change(screen.getByLabelText(/artist url/i), { target: { value: 'https://ligatures.example' } })
    fireEvent.click(screen.getByRole('button', { name: /^submit$/i }))

    await waitFor(() => expect(screen.getByText(/indexed ligatures/i)).toBeInTheDocument())
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('submitManifestUrl'), { url: 'https://ligatures.example' }, 'tok')
  })

  it('shows submission errors when the manifest is rejected', async () => {
    vi.spyOn(adminClient, 'adminGraphqlRequest').mockResolvedValue({
      submitManifestUrl: { accepted: false, errors: ['manifest failed validation: bad signature'], identity: null },
    })
    render(<AdminConsole />)

    fireEvent.change(screen.getByLabelText(/artist url/i), { target: { value: 'https://bad.example' } })
    fireEvent.click(screen.getByRole('button', { name: /^submit$/i }))

    await waitFor(() => expect(screen.getByText(/bad signature/i)).toBeInTheDocument())
  })

  it('shows a network/auth error from a rejected submit request', async () => {
    vi.spyOn(adminClient, 'adminGraphqlRequest').mockRejectedValue(new Error('Unauthorized: check the admin token.'))
    render(<AdminConsole />)

    fireEvent.change(screen.getByLabelText(/artist url/i), { target: { value: 'https://x.example' } })
    fireEvent.click(screen.getByRole('button', { name: /^submit$/i }))

    await waitFor(() => expect(screen.getByText(/unauthorized/i)).toBeInTheDocument())
  })

  it('validates pasted JSON and shows a valid result', async () => {
    vi.spyOn(adminClient, 'adminGraphqlRequest').mockResolvedValue({
      validateManifest: { valid: true, errors: [] },
    })
    render(<AdminConsole />)

    fireEvent.change(screen.getByLabelText(/manifest json/i), { target: { value: '{"manifestVersion":"1.0"}' } })
    fireEvent.click(screen.getByRole('button', { name: /^validate$/i }))

    await waitFor(() => expect(screen.getByText(/^valid\.$/i)).toBeInTheDocument())
  })

  it('shows field-level errors for an invalid pasted manifest', async () => {
    vi.spyOn(adminClient, 'adminGraphqlRequest').mockResolvedValue({
      validateManifest: { valid: false, errors: [{ field: 'identity.contactEmail', message: 'is required' }] },
    })
    render(<AdminConsole />)

    fireEvent.change(screen.getByLabelText(/manifest json/i), { target: { value: '{}' } })
    fireEvent.click(screen.getByRole('button', { name: /^validate$/i }))

    await waitFor(() => expect(screen.getByText(/identity.contactEmail: is required/i)).toBeInTheDocument())
  })

  it('loads an uploaded file into the textarea', async () => {
    render(<AdminConsole />)
    const file = new File(['{"manifestVersion":"1.0"}'], 'manifest.json', { type: 'application/json' })

    fireEvent.change(screen.getByLabelText(/upload manifest file/i), { target: { files: [file] } })

    await waitFor(() => expect(screen.getByLabelText(/manifest json/i)).toHaveValue('{"manifestVersion":"1.0"}'))
  })
})
