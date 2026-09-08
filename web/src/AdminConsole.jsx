import { useState, useEffect } from 'react'
import { adminGraphqlRequest, ADMIN_TOKEN_STORAGE_KEY } from './adminClient'

const SUBMIT_MUTATION = `
  mutation SubmitManifestUrl($url: String!) {
    submitManifestUrl(url: $url) {
      accepted
      errors
      identity { name type url }
    }
  }
`

const VALIDATE_MUTATION = `
  mutation ValidateManifest($rawJson: String!) {
    validateManifest(rawJson: $rawJson) {
      valid
      errors { field message }
    }
  }
`

function readStoredToken() {
  try {
    return sessionStorage.getItem(ADMIN_TOKEN_STORAGE_KEY) || ''
  } catch {
    return ''
  }
}

// The admin console for onboarding artists, per KB/0010-mvp-scope.md.
// Deliberately not linked from BrowseList/AlbumPage or anywhere else
// public: reaching it means typing /admin directly. The token is kept in
// sessionStorage only (cleared when the tab closes), never sent anywhere
// but the admin API itself.
export default function AdminConsole() {
  const [token, setToken] = useState(readStoredToken)

  useEffect(() => {
    try {
      sessionStorage.setItem(ADMIN_TOKEN_STORAGE_KEY, token)
    } catch {
      // sessionStorage can be unavailable (private browsing); the token
      // then just needs re-entering next time, no functional loss.
    }
  }, [token])

  return (
    <div className="admin-console">
      <h1>Admin console</h1>
      <p className="admin-note">Not linked from any public page. Changes here talk directly to the crawler.</p>

      <label className="admin-field">
        Admin token
        <input
          type="password"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder="paste ADMIN_TOKEN"
          aria-label="Admin token"
        />
      </label>

      <SubmitByUrl token={token} />
      <ValidateByHand token={token} />
    </div>
  )
}

function SubmitByUrl({ token }) {
  const [url, setUrl] = useState('')
  const [status, setStatus] = useState(null)

  async function handleSubmit(e) {
    e.preventDefault()
    setStatus({ pending: true })
    try {
      const data = await adminGraphqlRequest(SUBMIT_MUTATION, { url }, token)
      setStatus({ pending: false, result: data.submitManifestUrl })
    } catch (err) {
      setStatus({ pending: false, error: err.message })
    }
  }

  return (
    <section className="admin-section">
      <h2>Add artist by URL</h2>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://artist-site.example"
          aria-label="Artist URL"
          required
        />
        <button type="submit" disabled={status?.pending}>
          {status?.pending ? 'Submitting...' : 'Submit'}
        </button>
      </form>
      {status?.error && <p className="admin-error">{status.error}</p>}
      {status?.result &&
        (status.result.accepted ? (
          <p className="admin-success">Indexed {status.result.identity?.name}.</p>
        ) : (
          <ul className="admin-error">
            {status.result.errors.map((e, i) => (
              <li key={i}>{e}</li>
            ))}
          </ul>
        ))}
    </section>
  )
}

function ValidateByHand({ token }) {
  const [rawJson, setRawJson] = useState('')
  const [status, setStatus] = useState(null)

  function handleFile(e) {
    const file = e.target.files[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => setRawJson(String(reader.result))
    reader.readAsText(file)
  }

  async function handleValidate(e) {
    e.preventDefault()
    setStatus({ pending: true })
    try {
      const data = await adminGraphqlRequest(VALIDATE_MUTATION, { rawJson }, token)
      setStatus({ pending: false, result: data.validateManifest })
    } catch (err) {
      setStatus({ pending: false, error: err.message })
    }
  }

  return (
    <section className="admin-section">
      <h2>Validate a manifest by hand or upload</h2>
      <p className="admin-note">
        Dry-run only: checks structure and signature, never adds anything to the crawler's poll list.
      </p>
      <form onSubmit={handleValidate}>
        <input type="file" accept="application/json" onChange={handleFile} aria-label="Upload manifest file" />
        <textarea
          value={rawJson}
          onChange={(e) => setRawJson(e.target.value)}
          placeholder="Paste manifest.json contents here"
          aria-label="Manifest JSON"
          rows={10}
          required
        />
        <button type="submit" disabled={status?.pending}>
          {status?.pending ? 'Validating...' : 'Validate'}
        </button>
      </form>
      {status?.error && <p className="admin-error">{status.error}</p>}
      {status?.result &&
        (status.result.valid ? (
          <p className="admin-success">Valid.</p>
        ) : (
          <ul className="admin-error">
            {status.result.errors.map((e, i) => (
              <li key={i}>
                {e.field}: {e.message}
              </li>
            ))}
          </ul>
        ))}
    </section>
  )
}
