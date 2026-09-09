import { useState, useRef, useEffect, useCallback } from 'react'
import './AlbumPlayer.css'

function formatTime(s) {
  if (!s || isNaN(s)) return '0:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${sec.toString().padStart(2, '0')}`
}

function renderCreditParagraph(para, index) {
  if (typeof para === 'string') {
    return <p key={index}>{para}</p>
  }

  // Array-of-segments format: each segment is { text } or { text, url }
  if (Array.isArray(para)) {
    return (
      <p key={index}>
        {para.map((seg, i) =>
          seg.url ? (
            <a key={i} href={seg.url} target="_blank" rel="noopener noreferrer">
              {seg.text}
            </a>
          ) : (
            <span key={i}>{seg.text}</span>
          )
        )}
      </p>
    )
  }

  return null
}

// Social link icon components
const ICONS = {
  github: () => (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
    </svg>
  ),
  instagram: () => (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zM12 0C8.741 0 8.333.014 7.053.072 2.695.272.273 2.69.073 7.052.014 8.333 0 8.741 0 12c0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98C8.333 23.986 8.741 24 12 24c3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98C15.668.014 15.259 0 12 0zm0 5.838a6.162 6.162 0 100 12.324 6.162 6.162 0 000-12.324zM12 16a4 4 0 110-8 4 4 0 010 8zm6.406-11.845a1.44 1.44 0 100 2.881 1.44 1.44 0 000-2.881z"/>
    </svg>
  ),
  bandcamp: () => (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M0 18.75l7.437-13.5H24l-7.438 13.5z"/>
    </svg>
  ),
  spotify: () => (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 0C5.4 0 0 5.4 0 12s5.4 12 12 12 12-5.4 12-12S18.66 0 12 0zm5.521 17.34c-.24.359-.66.48-1.021.24-2.82-1.74-6.36-2.101-10.561-1.141-.418.122-.779-.179-.899-.539-.12-.421.18-.78.54-.9 4.56-1.021 8.52-.6 11.64 1.32.42.18.479.659.301 1.02zm1.44-3.3c-.301.42-.841.6-1.262.3-3.239-1.98-8.159-2.58-11.939-1.38-.479.12-1.02-.12-1.14-.6-.12-.48.12-1.021.6-1.141C9.6 9.9 15 10.561 18.72 12.84c.361.181.54.78.241 1.2zm.12-3.36C15.24 8.4 8.82 8.16 5.16 9.301c-.6.179-1.2-.181-1.38-.721-.18-.601.18-1.2.72-1.381 4.26-1.26 11.28-1.02 15.721 1.621.539.3.719 1.02.419 1.56-.299.421-1.02.599-1.559.3z"/>
    </svg>
  ),
  youtube: () => (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z"/>
    </svg>
  ),
  website: () => (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
    </svg>
  ),
}

// AlbumPlayer renders the full player, tracklist, lyrics viewer, and
// social/download links for one album, given `config` in the shape
// adapters.js's albumToPlayerConfig produces. Adapted from
// https://github.com/geekpunk/albumn-page-generator's App.jsx per
// KB/0010-mvp-scope.md: that component originally self-fetched a static
// album-config.json and prefixed asset paths with a build-time BASE_URL.
// Here config always arrives as a prop (AlbumPage builds it from a GraphQL
// response), and every asset path (images, track files, the ZIP download)
// is already an absolute, artist-hosted URL per KB/0003-manifest.md, so
// this version uses them directly instead of prefixing anything.
function AlbumPlayer({ config }) {
  const [showLyrics, setShowLyrics] = useState(false)
  const [flipped, setFlipped] = useState(false)
  const [currentTrack, setCurrentTrack] = useState(0)
  const [isPlaying, setIsPlaying] = useState(false)
  const [progress, setProgress] = useState(0)
  const [duration, setDuration] = useState(0)
  const audioRef = useRef(null)

  useEffect(() => {
    document.title = config.pageTitle

    if (config.colors) {
      const root = document.documentElement
      if (config.colors.primary) root.style.setProperty('--color-primary', config.colors.primary)
      if (config.colors.heroBackground) root.style.setProperty('--color-hero-bg', config.colors.heroBackground)
      if (config.colors.background) root.style.setProperty('--color-bg', config.colors.background)
      if (config.colors.text) root.style.setProperty('--color-text', config.colors.text)
    }

    const params = new URLSearchParams(window.location.search)
    const song = params.get('song')
    if (song) {
      const idx = config.tracks.findIndex((t) => t.number.toLowerCase() === song.toLowerCase())
      if (idx !== -1) setCurrentTrack(idx)
    }
  }, [config])

  const tracks = config.tracks || []
  const sides = [...new Set(tracks.map((t) => t.side))]

  const updateQueryString = useCallback((index) => {
    if (!tracks[index]) return
    const url = new URL(window.location)
    url.searchParams.set('song', tracks[index].number)
    window.history.replaceState({}, '', url)
  }, [tracks])

  const playTrack = useCallback((index) => {
    if (!tracks[index]) return
    setCurrentTrack(index)
    setIsPlaying(true)
    updateQueryString(index)
    if (audioRef.current) {
      audioRef.current.src = tracks[index].file
      audioRef.current.play().catch(() => {})
    }
  }, [updateQueryString, tracks])

  const togglePlay = () => {
    if (!audioRef.current || !tracks[currentTrack]) return
    if (isPlaying) {
      audioRef.current.pause()
      setIsPlaying(false)
    } else {
      if (!audioRef.current.src || audioRef.current.src === window.location.href) {
        audioRef.current.src = tracks[currentTrack].file
      }
      audioRef.current.play().catch(() => {})
      setIsPlaying(true)
    }
  }

  const prevTrack = () => {
    const prev = currentTrack === 0 ? tracks.length - 1 : currentTrack - 1
    playTrack(prev)
  }

  const nextTrack = useCallback(() => {
    if (tracks.length === 0) return
    const next = (currentTrack + 1) % tracks.length
    playTrack(next)
  }, [currentTrack, playTrack, tracks.length])

  const handleSeek = (e) => {
    if (!audioRef.current || !duration) return
    const rect = e.currentTarget.getBoundingClientRect()
    const x = e.clientX - rect.left
    const pct = x / rect.width
    audioRef.current.currentTime = pct * duration
  }

  useEffect(() => {
    const audio = audioRef.current
    if (!audio) return

    const onTimeUpdate = () => setProgress(audio.currentTime)
    const onDurationChange = () => setDuration(audio.duration)
    const onEnded = () => nextTrack()

    audio.addEventListener('timeupdate', onTimeUpdate)
    audio.addEventListener('durationchange', onDurationChange)
    audio.addEventListener('ended', onEnded)

    return () => {
      audio.removeEventListener('timeupdate', onTimeUpdate)
      audio.removeEventListener('durationchange', onDurationChange)
      audio.removeEventListener('ended', onEnded)
    }
  }, [nextTrack])

  const pct = duration ? (progress / duration) * 100 : 0

  const linkEntries = config.links ? Object.entries(config.links).filter(([type]) => ICONS[type]) : []

  return (
    <div className="site">
      <audio ref={audioRef} preload="metadata" />

      <section className="hero">
        {config.images.back ? (
          <div className="album-flip-container" onClick={() => setFlipped(!flipped)}>
            <div className={`album-flip-inner${flipped ? ' flipped' : ''}`}>
              <div className="album-flip-face album-flip-front">
                <img
                  src={config.images.front}
                  alt={`${config.artistName} - ${config.albumName} album cover`}
                />
              </div>
              <div className="album-flip-face album-flip-back">
                <img
                  src={config.images.back}
                  alt={`${config.artistName} - ${config.albumName} album back cover`}
                />
              </div>
            </div>
            <div className="flip-hint">{flipped ? 'front' : 'back'}</div>
          </div>
        ) : (
          // No back cover declared (KB/0003-manifest.md's images.back is
          // optional): show the front cover only, not flippable, rather
          // than a flip control whose back face has no image to show.
          <div className="album-flip-container">
            <img
              src={config.images.front}
              alt={`${config.artistName} - ${config.albumName} album cover`}
            />
          </div>
        )}

        <h1 className="album-title">
          {config.artistName} <span>&ndash; {config.albumName}</span>
        </h1>

        <div className="player">
          <div className="player-now-playing">
            {tracks[currentTrack].number}. {tracks[currentTrack].name}
          </div>
          <div className="player-controls">
            <button className="player-btn" onClick={prevTrack} aria-label="Previous track">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M6 6h2v12H6zm3.5 6l8.5 6V6z"/>
              </svg>
            </button>
            <button className="player-btn player-btn-play" onClick={togglePlay} aria-label={isPlaying ? 'Pause' : 'Play'}>
              {isPlaying ? (
                <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>
                </svg>
              ) : (
                <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M8 5v14l11-7z"/>
                </svg>
              )}
            </button>
            <button className="player-btn" onClick={nextTrack} aria-label="Next track">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M6 18l8.5-6L6 6v12zM16 6v12h2V6h-2z"/>
              </svg>
            </button>
          </div>
          <div className="player-progress" onClick={handleSeek}>
            <div className="player-progress-bar" style={{ width: `${pct}%` }} />
          </div>
          <div className="player-time">
            <span>{formatTime(progress)}</span>
            <span>{formatTime(duration)}</span>
          </div>
          {tracks[currentTrack]?.lyrics && (
            <button
              className="player-btn-lyrics"
              onClick={() => setShowLyrics(true)}
            >
              Lyrics
            </button>
          )}
        </div>

        {config.downloadZip && (
          <div className="album-download">
            <a
              className="album-download-btn"
              href={config.downloadZip}
              download
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" style={{ marginRight: '8px' }}>
                <path d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z"/>
              </svg>
              Download Full Album (ZIP)
            </a>
          </div>
        )}
      </section>

      {showLyrics && tracks[currentTrack]?.lyrics && (
        <div className="lyrics-overlay" onClick={() => setShowLyrics(false)}>
          <div className="lyrics-dialog" onClick={(e) => e.stopPropagation()}>
            <div className="lyrics-header">
              <h3>{tracks[currentTrack].name}</h3>
              <button
                className="lyrics-close"
                onClick={() => setShowLyrics(false)}
                aria-label="Close lyrics"
              >
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
                </svg>
              </button>
            </div>
            <pre className="lyrics-body">{tracks[currentTrack].lyrics}</pre>
          </div>
        </div>
      )}

      <section className="tracks">
        <h2>Tracklist</h2>

        {sides.map((side) => (
          <div key={side}>
            <div className="side-label">Side {side}</div>
            {tracks
              .map((track, idx) => ({ track, idx }))
              .filter(({ track }) => track.side === side)
              .map(({ track, idx }) => {
                const active = idx === currentTrack
                return (
                  <div
                    className={`track${active ? ' track-active' : ''}`}
                    key={track.number}
                    onClick={() => playTrack(idx)}
                  >
                    <span className="track-number">
                      {active && isPlaying ? (
                        <span className="eq-bars">
                          <span /><span /><span />
                        </span>
                      ) : (
                        track.number
                      )}
                    </span>
                    <span className="track-name">{track.name}</span>
                    <span className="track-duration">{track.duration}</span>
                    <a
                      className="track-download"
                      href={track.file}
                      download
                      onClick={(e) => e.stopPropagation()}
                    >
                      Download
                    </a>
                  </div>
                )
              })}
          </div>
        ))}
      </section>

      {config.credits && (
        <section className="credits">
          <h2>Credits</h2>
          <div className="credits-text">
            {config.credits.paragraphs.map((para, index) => renderCreditParagraph(para, index))}
          </div>
        </section>
      )}

      {linkEntries.length > 0 && (
        <nav className="links">
          {linkEntries.map(([type, url]) => {
            const Icon = ICONS[type]
            return (
              <a key={type} href={url} target="_blank" rel="noopener noreferrer" aria-label={type}>
                <Icon />
              </a>
            )
          })}
        </nav>
      )}

      {config.footer && <footer className="footer">{config.footer}</footer>}
    </div>
  )
}

export default AlbumPlayer
