import { BrowserRouter, Routes, Route } from 'react-router-dom'
import BrowseList from './BrowseList'
import AlbumPage from './AlbumPage'
import AdminConsole from './AdminConsole'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<BrowseList />} />
        <Route path="/album/:identityUrl/:albumId" element={<AlbumPage />} />
        {/* Deliberately not linked from anywhere above, per
            KB/0010-mvp-scope.md's admin console design. */}
        <Route path="/admin" element={<AdminConsole />} />
      </Routes>
    </BrowserRouter>
  )
}
