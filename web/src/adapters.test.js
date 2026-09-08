import { describe, it, expect } from 'vitest'
import { albumToPlayerConfig, defaultColors } from './adapters'

function baseAlbum(overrides = {}) {
  return {
    albumName: 'Agency',
    pageTitle: null,
    downloadZip: null,
    credits: null,
    imagesFront: 'https://ligatures.example/front.png',
    imagesBack: 'https://ligatures.example/back.png',
    identity: { name: 'Ligatures' },
    presentation: null,
    tracks: [
      { number: 'A1', side: 'A', name: 'Opening', duration: "3'30\"", file: 'https://ligatures.example/opening.mp3', lyrics: null },
    ],
    ...overrides,
  }
}

describe('albumToPlayerConfig', () => {
  it('maps core fields directly', () => {
    const config = albumToPlayerConfig(baseAlbum())
    expect(config.albumName).toBe('Agency')
    expect(config.artistName).toBe('Ligatures')
    expect(config.images).toEqual({
      front: 'https://ligatures.example/front.png',
      back: 'https://ligatures.example/back.png',
    })
    expect(config.tracks).toHaveLength(1)
    expect(config.tracks[0]).toMatchObject({ number: 'A1', name: 'Opening', file: 'https://ligatures.example/opening.mp3' })
  })

  it('derives pageTitle from artist and album name when absent', () => {
    const config = albumToPlayerConfig(baseAlbum({ pageTitle: null }))
    expect(config.pageTitle).toBe('Ligatures - Agency')
  })

  it('keeps an explicit pageTitle', () => {
    const config = albumToPlayerConfig(baseAlbum({ pageTitle: 'Custom Title' }))
    expect(config.pageTitle).toBe('Custom Title')
  })

  it('falls back to default colors when no presentation is set', () => {
    const config = albumToPlayerConfig(baseAlbum({ presentation: null }))
    expect(config.colors).toEqual(defaultColors)
  })

  it('uses presentation colors when present', () => {
    const config = albumToPlayerConfig(
      baseAlbum({
        presentation: { colors: { primary: '#111', heroBackground: '#222', background: '#333', text: '#444' }, links: { bandcamp: 'https://x' }, footer: 'hi' },
      })
    )
    expect(config.colors).toEqual({ primary: '#111', heroBackground: '#222', background: '#333', text: '#444' })
    expect(config.links).toEqual({ bandcamp: 'https://x' })
    expect(config.footer).toBe('hi')
  })

  it('defaults links to an empty object and footer to an empty string when absent', () => {
    const config = albumToPlayerConfig(baseAlbum({ presentation: null }))
    expect(config.links).toEqual({})
    expect(config.footer).toBe('')
  })
})
