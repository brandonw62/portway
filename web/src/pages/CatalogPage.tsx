import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { listResourceTypes, type ResourceType } from '../api';

export default function CatalogPage() {
  const [types, setTypes] = useState<ResourceType[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    listResourceTypes().then(setTypes).catch((e: Error) => setError(e.message));
  }, []);

  return (
    <>
      <h2>Resource Catalog</h2>
      {error && <div className="error">{error}</div>}
      {types.length === 0 && !error && (
        <div className="empty">No resource types available yet.</div>
      )}
      <div className="card-grid">
        {types.map((t) => (
          <Link to={`/resources/new?type=${t.id}`} key={t.id} style={{ textDecoration: 'none', color: 'inherit' }}>
            <div className="card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
                <h3>{t.name}</h3>
                <span className={`badge ${t.category}`}>{t.category}</span>
              </div>
              <p>{t.description || 'No description'}</p>
            </div>
          </Link>
        ))}
      </div>
    </>
  );
}
