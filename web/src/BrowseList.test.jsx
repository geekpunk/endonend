import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import BrowseList from './BrowseList'
import * as client from './graphqlClient'

afterEach(() => {
  vi.restoreAllMocks()
})

function renderBrowseList() {
  return render(
    <MemoryRouter>
      <BrowseList />
    </MemoryRouter>
  )
}

describe('BrowseList', () => {
  it('shows a loading state, then the albums once loaded', async () => {
    vi.spyOn(client, 'graphqlRequest').mockResolvedValue({
      albums: [
        { albumId: 'agency-2024', albumName: 'Agency', releaseDate: '2024-05-01', imagesFront: 'https://x/front.png', identity: { url: 'https://ligatures.example', name: 'Ligatures' } },
      ],
    })

    renderBrowseList()

    expect(screen.getByText(/loading/i)).toBeInTheDocument()

    await waitFor(() => expect(screen.getByText('Agency')).toBeInTheDocument())
    expect(screen.getByText('Ligatures')).toBeInTheDocument()

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', `/album/${encodeURIComponent('https://ligatures.example')}/agency-2024`)
  })

  it('shows an empty state when there are no albums', async () => {
    vi.spyOn(client, 'graphqlRequest').mockResolvedValue({ albums: [] })
    renderBrowseList()
    await waitFor(() => expect(screen.getByText(/no albums indexed yet/i)).toBeInTheDocument())
  })

  it('shows an error state when the request fails', async () => {
    vi.spyOn(client, 'graphqlRequest').mockRejectedValue(new Error('network down'))
    renderBrowseList()
    await waitFor(() => expect(screen.getByText(/network down/i)).toBeInTheDocument())
  })
})
