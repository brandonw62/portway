import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { listResourceTypes, createResource, type ResourceType } from '../api';
import { useWorkspace } from '../WorkspaceContext';

export default function ProvisionPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const preselectedType = searchParams.get('type') || '';
  const { activeProject } = useWorkspace();

  const [types, setTypes] = useState<ResourceType[]>([]);
  const [typeId, setTypeId] = useState(preselectedType);
  const [name, setName] = useState('');
  const [specJson, setSpecJson] = useState('{}');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    listResourceTypes().then((t) => {
      setTypes(t);
      // Apply default spec when type is preselected
      if (preselectedType) {
        const selected = t.find((rt) => rt.id === preselectedType);
        if (selected?.default_spec) {
          setSpecJson(JSON.stringify(selected.default_spec, null, 2));
        }
      }
    }).catch((e: Error) => setError(e.message));
  }, [preselectedType]);

  const handleTypeChange = (id: string) => {
    setTypeId(id);
    const selected = types.find((t) => t.id === id);
    if (selected?.default_spec) {
      setSpecJson(JSON.stringify(selected.default_spec, null, 2));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!typeId || !name.trim()) {
      setError('Resource type and name are required');
      return;
    }

    let spec: Record<string, unknown>;
    try {
      spec = JSON.parse(specJson);
    } catch {
      setError('Spec must be valid JSON');
      return;
    }

    setSubmitting(true);
    try {
      if (!activeProject) {
        setError('Select a project first');
        setSubmitting(false);
        return;
      }
      await createResource({
        project_id: activeProject.id,
        resource_type_id: typeId,
        name: name.trim(),
        spec,
      });
      navigate('/resources');
    } catch (e) {
      setError((e as Error).message);
      setSubmitting(false);
    }
  };

  return (
    <>
      <h2>Provision New Resource</h2>
      {error && <div className="error">{error}</div>}
      <form onSubmit={handleSubmit} style={{ maxWidth: 500 }}>
        <div className="form-group">
          <label>Resource Type</label>
          <select value={typeId} onChange={(e) => handleTypeChange(e.target.value)}>
            <option value="">Select a type...</option>
            {types.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name} ({t.category})
              </option>
            ))}
          </select>
        </div>
        <div className="form-group">
          <label>Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. my-postgres-db"
          />
        </div>
        <div className="form-group">
          <label>Configuration (JSON)</label>
          <textarea
            rows={8}
            value={specJson}
            onChange={(e) => setSpecJson(e.target.value)}
            style={{ fontFamily: 'monospace' }}
          />
        </div>
        <div className="actions">
          <button className="btn btn-secondary" type="button" onClick={() => navigate(-1)}>
            Cancel
          </button>
          <button className="btn btn-primary" type="submit" disabled={submitting}>
            {submitting ? 'Provisioning...' : 'Provision Resource'}
          </button>
        </div>
      </form>
    </>
  );
}
