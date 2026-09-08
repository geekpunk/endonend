import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { graphqlRequest } from './graphqlClient'

const ALBUMS_QUERY = `
  query Albums {
    albums(limit: 100) {
      albumId
      albumName
      releaseDate
      imagesFront
      identity { url name }
    }
  }
`

// The MVP browse view, per KB/0010-mvp-scope.md: a plain list linking to
// each album's playback page, no search engine.
export default function BrowseList() {
  const [state, setState] = useState({ loading: true, error: null, albums: [] })

  useEffect(() => {
    let cancelled = false
    graphqlRequest(ALBUMS_QUERY)
      .then((data) => {
        if (!cancelled) setState({ loading: false, error: null, albums: data.albums })
      })
      .catch((err) => {
        if (!cancelled) setState({ loading: false, error: err.message, albums: [] })
      })
    return () => {
      cancelled = true
    }
  }, [])

  if (state.loading) {
    return <div className="page-status">Loading...</div>
  }
  if (state.error) {
    return <div className="page-status">Couldn't load albums: {state.error}</div>
  }
  if (state.albums.length === 0) {
    return <div className="page-status">No albums indexed yet.</div>
  }

  return (
    <ul className="browse-list">
      {state.albums.map((album) => (
        <li key={`${album.identity.url}/${album.albumId}`} className="browse-list-item">
          <Link
            to={`/album/${encodeURIComponent(album.identity.url)}/${encodeURIComponent(album.albumId)}`}
          >
            <img src={album.imagesFront} alt="" className="browse-list-thumb" />
            <div>
              <div className="browse-list-album">{album.albumName}</div>
              <div className="browse-list-artist">{album.identity.name}</div>
            </div>
          </Link>
        </li>
      ))}
    </ul>
  )
}
