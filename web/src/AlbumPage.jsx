import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import AlbumPlayer from './AlbumPlayer'
import { albumToPlayerConfig } from './adapters'
import { graphqlRequest } from './graphqlClient'

const ALBUM_QUERY = `
  query Album($identityUrl: String!, $albumId: String!) {
    album(identityUrl: $identityUrl, albumId: $albumId) {
      albumName
      pageTitle
      downloadZip
      credits
      imagesFront
      imagesBack
      identity { name }
      presentation { colors { primary heroBackground background text } links footer }
      tracks { number side name duration file lyrics }
    }
  }
`

// Route params carry identityUrl/albumId URL-encoded (per App.jsx's Link
// hrefs), since identityUrl is itself a full URL, per KB/0003-manifest.md.
export default function AlbumPage() {
  const { identityUrl, albumId } = useParams()
  const [state, setState] = useState({ loading: true, error: null, config: null })

  useEffect(() => {
    let cancelled = false
    setState({ loading: true, error: null, config: null })

    graphqlRequest(ALBUM_QUERY, {
      identityUrl: decodeURIComponent(identityUrl),
      albumId: decodeURIComponent(albumId),
    })
      .then((data) => {
        if (cancelled) return
        if (!data.album) {
          setState({ loading: false, error: 'Album not found.', config: null })
          return
        }
        setState({ loading: false, error: null, config: albumToPlayerConfig(data.album) })
      })
      .catch((err) => {
        if (cancelled) return
        setState({ loading: false, error: err.message, config: null })
      })

    return () => {
      cancelled = true
    }
  }, [identityUrl, albumId])

  if (state.loading) {
    return <div className="page-status">Loading...</div>
  }
  if (state.error) {
    return <div className="page-status">Couldn't load this album: {state.error}</div>
  }
  return <AlbumPlayer config={state.config} />
}
