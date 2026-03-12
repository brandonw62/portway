import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { listApprovals, type ApprovalRequest } from '../api';

export default function ApprovalListPage() {
  const [approvals, setApprovals] = useState<ApprovalRequest[]>([]);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    listApprovals().then(setApprovals).catch((e: Error) => setError(e.message));
  }, []);

  return (
    <>
      <div className="page-header">
        <h2>Pending Approvals</h2>
      </div>
      {error && <div className="error">{error}</div>}
      {approvals.length === 0 && !error ? (
        <div className="empty">No pending approvals.</div>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Resource</th>
              <th>Type</th>
              <th>Requested By</th>
              <th>Policy Reasons</th>
              <th>Requested</th>
              <th>Expires</th>
            </tr>
          </thead>
          <tbody>
            {approvals.map((a) => {
              const resourceName =
                (a.request_payload as Record<string, unknown>)?.resource_id as string || a.id;
              return (
                <tr key={a.id} onClick={() => navigate(`/approvals/${a.id}`)}>
                  <td>{resourceName}</td>
                  <td>{a.resource_type}</td>
                  <td>{a.requested_by}</td>
                  <td>
                    {Array.isArray(a.reasons)
                      ? a.reasons.join(', ')
                      : '-'}
                  </td>
                  <td style={{ color: 'var(--text-muted)' }}>
                    {new Date(a.created_at).toLocaleDateString()}
                  </td>
                  <td style={{ color: 'var(--text-muted)' }}>
                    {new Date(a.expires_at).toLocaleDateString()}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </>
  );
}
