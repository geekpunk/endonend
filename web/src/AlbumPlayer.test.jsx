import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import AlbumPlayer from './AlbumPlayer'

function config(overrides = {}) {
  return {
    albumName: 'Agency',
    artistName: 'Ligatures',
    pageTitle: 'Ligatures - Agency',
    images: { front: 'https://ligatures.example/front.png', back: 'https://ligatures.example/back.png' },
    colors: { primary: '#5B96C7', heroBackground: '#5B96C7', background: '#1a1a1a', text: '#e0e0e0' },
    downloadZip: 'https://ligatures.example/agency.zip',
    credits: null,
    links: {},
    footer: '',
    tracks: [
      { number: 'A1', side: 'A', name: 'Opening', duration: "3'30\"", file: 'https://ligatures.example/opening.mp3', lyrics: 'First line\nSecond line' },
      { number: 'A2', side: 'A', name: 'Second Song', duration: "2'45\"", file: 'https://ligatures.example/second.mp3', lyrics: null },
    ],
    ...overrides,
  }
}

beforeEach(() => {
  window.HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined)
  window.HTMLMediaElement.prototype.pause = vi.fn()
  window.history.replaceState({}, '', '/')
})

describe('AlbumPlayer', () => {
  it('renders the album title, now-playing track, and tracklist using absolute file URLs directly', () => {
    render(<AlbumPlayer config={config()} />)

    expect(screen.getByRole('heading', { name: /Ligatures.*Agency/ })).toBeInTheDocument()
    expect(screen.getByText(/A1\. Opening/)).toBeInTheDocument()
    expect(screen.getByText('Second Song')).toBeInTheDocument()

    // The vendored template originally prefixed these with a build-time
    // BASE_URL; the adapted version must use the already-absolute URLs
    // the manifest provides as-is (see AlbumPlayer.jsx's file header note).
    const images = screen.getAllByRole('img')
    expect(images[0]).toHaveAttribute('src', 'https://ligatures.example/front.png')
    const zipLink = screen.getByRole('link', { name: /download full album/i })
    expect(zipLink).toHaveAttribute('href', 'https://ligatures.example/agency.zip')
  })

  it('plays the clicked track and updates the now-playing label', () => {
    render(<AlbumPlayer config={config()} />)

    fireEvent.click(screen.getByText('Second Song'))

    expect(screen.getByText(/A2\. Second Song/)).toBeInTheDocument()
    expect(window.HTMLMediaElement.prototype.play).toHaveBeenCalled()
  })

  it('toggles play/pause on the main control', () => {
    render(<AlbumPlayer config={config()} />)

    fireEvent.click(screen.getByRole('button', { name: /^play$/i }))
    expect(window.HTMLMediaElement.prototype.play).toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: /^pause$/i }))
    expect(window.HTMLMediaElement.prototype.pause).toHaveBeenCalled()
  })

  it('shows lyrics for a track that has them and lets the overlay be closed', () => {
    render(<AlbumPlayer config={config()} />)

    fireEvent.click(screen.getByRole('button', { name: /lyrics/i }))
    expect(screen.getByText(/First line/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /close lyrics/i }))
    expect(screen.queryByText(/First line/)).not.toBeInTheDocument()
  })

  it('does not show a lyrics button for a track with no lyrics', () => {
    render(<AlbumPlayer config={config()} />)
    fireEvent.click(screen.getByText('Second Song'))
    expect(screen.queryByRole('button', { name: /^lyrics$/i })).not.toBeInTheDocument()
  })

  it('links each track download directly to its absolute file URL', () => {
    render(<AlbumPlayer config={config()} />)
    const downloadLinks = screen.getAllByRole('link', { name: /download/i }).filter((l) => l.textContent === 'Download')
    expect(downloadLinks[0]).toHaveAttribute('href', 'https://ligatures.example/opening.mp3')
  })

  it('flips the album art between front and back on click', () => {
    render(<AlbumPlayer config={config()} />)
    const images = screen.getAllByRole('img')
    const flipContainer = images[0].closest('.album-flip-container')

    expect(screen.getByText('back')).toBeInTheDocument()
    fireEvent.click(flipContainer)
    expect(screen.getByText('front')).toBeInTheDocument()
  })

  it('shows the front cover only, with no flip control, when there is no back cover', () => {
    render(<AlbumPlayer config={config({ images: { front: 'https://ligatures.example/front.png', back: null } })} />)

    const images = screen.getAllByRole('img')
    expect(images.filter((img) => img.alt.includes('cover'))).toHaveLength(1)
    expect(screen.queryByText('back')).not.toBeInTheDocument()
    expect(screen.queryByText('front')).not.toBeInTheDocument()
  })
})
