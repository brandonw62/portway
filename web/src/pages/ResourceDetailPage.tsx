import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getResource, deleteResource, type Resource } from '../api';

export default function ResourceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [resource, setResource] = useState<Resource | null>(null);
  const [error, setError] = useState('');
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (id) {
      getResource(id).then(setResource).catch((e: Error) => setError(e.message));
    }
  }, [id]);

  const handleDelete = async () => {
    if (!id || !confirm('Are you sure you want to delete this resource?')) return;
    setDeleting(true);
    try {
      await deleteResource(id);
      navigate('/resources');
    } catch (e) {
      setError((e as Error).message);
      setDeleting(false);
    }
  };

  if (error) return <div className="error">{error}</div>;
  if (!resource) return <div className="empty">Loading...</div>;

  return (
    <>
      <div className="page-header">
        <h2>{resource.name}</h2>
        <span className={`status ${resource.status}`}>{resource.status}</span>
      </div>
      {resource.status_message && (
        <div className="error" style={{ marginBottom: '1rem' }}>{resource.status_message}</div>
      )}
      <dl className="detail-grid">
        <dt>ID</dt>
        <dd>{resource.id}</dd>
        <dt>Slug</dt>
        <dd>{resource.slug}</dd>
        <dt>Status</dt>
        <dd><span className={`status ${resource.status}`}>{resource.status}</span></dd>
        <dt>Project ID</dt>
        <dd>{resource.project_id}</dd>
        <dt>Resource Type</dt>
        <dd>{resource.resource_type_id}</dd>
        <dt>Provider Ref</dt>
        <dd>{resource.provider_ref || '-'}</dd>
        <dt>Requested By</dt>
        <dd>{resource.requested_by}</dd>
        <dt>Created</dt>
        <dd>{new Date(resource.created_at).toLocaleString()}</dd>
        <dt>Updated</dt>
        <dd>{new Date(resource.updated_at).toLocaleString()}</dd>
        <dt>Spec</dt>
        <dd><pre style={{ whiteSpace: 'pre-wrap', fontSize: '0.8rem' }}>{JSON.stringify(resource.spec, null, 2)}</pre></dd>
      </dl>
      <div className="actions">
        <button className="btn btn-secondary" onClick={() => navigate('/resources')}>Back</button>
        {resource.status !== 'deleted' && resource.status !== 'deleting' && (
          <button className="btn btn-danger" onClick={handleDelete} disabled={deleting}>
            {deleting ? 'Deleting...' : 'Delete Resource'}
          </button>
        )}
      </div>
    </>
  );
}
