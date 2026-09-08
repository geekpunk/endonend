import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import App from './App'
import * as client from './graphqlClient'

afterEach(() => {
  vi.restoreAllMocks()
  window.history.replaceState({}, '', '/')
})

describe('App routing', () => {
  it('renders the browse list at the root path', async () => {
    vi.spyOn(client, 'graphqlRequest').mockResolvedValue({ albums: [] })
    render(<App />)
    await waitFor(() => expect(screen.getByText(/no albums indexed yet/i)).toBeInTheDocument())
  })
})
