// Maps a GraphQL `album` query result (per KB/0010-mvp-scope.md's API
// section) into the config shape the vendored album-page-generator
// component (AlbumPlayer.jsx) expects, per that repo's own
// "Config Reference". The two shapes are already close by design; this is
// the small, explicit translation KB/0010 calls for rather than assuming
// they're interchangeable.
export function albumToPlayerConfig(album) {
  return {
    albumName: album.albumName,
    artistName: album.identity.name,
    pageTitle: album.pageTitle || `${album.identity.name} - ${album.albumName}`,
    images: {
      front: album.imagesFront,
      back: album.imagesBack,
    },
    colors: album.presentation?.colors
      ? {
          primary: album.presentation.colors.primary,
          heroBackground: album.presentation.colors.heroBackground,
          background: album.presentation.colors.background,
          text: album.presentation.colors.text,
        }
      : defaultColors,
    downloadZip: album.downloadZip || null,
    credits: album.credits || null,
    links: album.presentation?.links || {},
    footer: album.presentation?.footer || '',
    tracks: album.tracks.map((t) => ({
      number: t.number,
      side: t.side || 'A',
      name: t.name,
      duration: t.duration,
      file: t.file,
      lyrics: t.lyrics || null,
    })),
  }
}

export const defaultColors = {
  primary: '#5B96C7',
  heroBackground: '#5B96C7',
  background: '#1a1a1a',
  text: '#e0e0e0',
}
