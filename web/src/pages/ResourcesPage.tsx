import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { listResources, type Resource } from '../api';

// Hardcoded project ID for now — will come from project switcher later
const PROJECT_ID = 'default-project';

export default function ResourcesPage() {
  const [resources, setResources] = useState<Resource[]>([]);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    listResources(PROJECT_ID).then(setResources).catch((e: Error) => setError(e.message));
  }, []);

  return (
    <>
      <div className="page-header">
        <h2>My Resources</h2>
        <button className="btn btn-primary" onClick={() => navigate('/resources/new')}>
          Provision New
        </button>
      </div>
      {error && <div className="error">{error}</div>}
      {resources.length === 0 && !error ? (
        <div className="empty">No resources provisioned yet. Browse the catalog to get started.</div>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Slug</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {resources.map((r) => (
              <tr key={r.id} onClick={() => navigate(`/resources/${r.id}`)}>
                <td>{r.name}</td>
                <td><span className={`status ${r.status}`}>{r.status}</span></td>
                <td style={{ color: 'var(--text-muted)' }}>{r.slug}</td>
                <td style={{ color: 'var(--text-muted)' }}>{new Date(r.created_at).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}
