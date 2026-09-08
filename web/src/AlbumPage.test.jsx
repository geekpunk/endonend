import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import AlbumPage from './AlbumPage'
import * as client from './graphqlClient'

afterEach(() => {
  vi.restoreAllMocks()
})

function renderAtRoute(path) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/album/:identityUrl/:albumId" element={<AlbumPage />} />
      </Routes>
    </MemoryRouter>
  )
}

const routePath = `/album/${encodeURIComponent('https://ligatures.example')}/${encodeURIComponent('agency-2024')}`

describe('AlbumPage', () => {
  it('requests the album using the decoded route params and renders the player', async () => {
    const spy = vi.spyOn(client, 'graphqlRequest').mockResolvedValue({
      album: {
        albumName: 'Agency',
        pageTitle: null,
        downloadZip: null,
        credits: null,
        imagesFront: 'https://ligatures.example/front.png',
        imagesBack: 'https://ligatures.example/back.png',
        identity: { name: 'Ligatures' },
        presentation: null,
        tracks: [{ number: 'A1', side: 'A', name: 'Opening', duration: "3'30\"", file: 'https://ligatures.example/opening.mp3', lyrics: null }],
      },
    })

    renderAtRoute(routePath)

    expect(screen.getByText(/loading/i)).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('Opening')).toBeInTheDocument())

    expect(spy).toHaveBeenCalledWith(
      expect.stringContaining('query Album'),
      { identityUrl: 'https://ligatures.example', albumId: 'agency-2024' }
    )
  })

  it('shows a not-found message when the album does not exist', async () => {
    vi.spyOn(client, 'graphqlRequest').mockResolvedValue({ album: null })
    renderAtRoute(routePath)
    await waitFor(() => expect(screen.getByText(/album not found/i)).toBeInTheDocument())
  })

  it('shows an error message when the request fails', async () => {
    vi.spyOn(client, 'graphqlRequest').mockRejectedValue(new Error('boom'))
    renderAtRoute(routePath)
    await waitFor(() => expect(screen.getByText(/boom/i)).toBeInTheDocument())
  })
})
