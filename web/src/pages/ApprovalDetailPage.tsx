import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getApproval, reviewApproval, type ApprovalRequest } from '../api';

export default function ApprovalDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [approval, setApproval] = useState<ApprovalRequest | null>(null);
  const [error, setError] = useState('');
  const [comment, setComment] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (id) {
      getApproval(id).then(setApproval).catch((e: Error) => setError(e.message));
    }
  }, [id]);

  const handleReview = async (decision: 'approved' | 'denied') => {
    if (!id) return;
    setSubmitting(true);
    try {
      const updated = await reviewApproval(id, { decision, comment });
      setApproval(updated);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  if (error) return <div className="error">{error}</div>;
  if (!approval) return <div className="empty">Loading...</div>;

  const isPending = approval.status === 'pending';
  const payload = approval.request_payload as Record<string, unknown>;

  return (
    <>
      <div className="page-header">
        <h2>Approval Request</h2>
        <span className={`status ${approval.status}`}>{approval.status}</span>
      </div>

      <dl className="detail-grid">
        <dt>ID</dt>
        <dd>{approval.id}</dd>
        <dt>Project ID</dt>
        <dd>{approval.project_id}</dd>
        <dt>Resource Type</dt>
        <dd>{approval.resource_type}</dd>
        <dt>Requested By</dt>
        <dd>{approval.requested_by}</dd>
        <dt>Created</dt>
        <dd>{new Date(approval.created_at).toLocaleString()}</dd>
        <dt>Expires</dt>
        <dd>{new Date(approval.expires_at).toLocaleString()}</dd>
        <dt>Status</dt>
        <dd><span className={`status ${approval.status}`}>{approval.status}</span></dd>

        {approval.reviewed_by && (
          <>
            <dt>Reviewed By</dt>
            <dd>{approval.reviewed_by}</dd>
          </>
        )}
        {approval.review_comment && (
          <>
            <dt>Review Comment</dt>
            <dd>{approval.review_comment}</dd>
          </>
        )}
        {approval.reviewed_at && (
          <>
            <dt>Reviewed At</dt>
            <dd>{new Date(approval.reviewed_at).toLocaleString()}</dd>
          </>
        )}

        <dt>Policy Reasons</dt>
        <dd>
          {Array.isArray(approval.reasons) && approval.reasons.length > 0 ? (
            <ul style={{ margin: 0, paddingLeft: '1.2rem' }}>
              {approval.reasons.map((r, i) => (
                <li key={i}>{r}</li>
              ))}
            </ul>
          ) : (
            '-'
          )}
        </dd>

        <dt>Matched Policies</dt>
        <dd>
          {Array.isArray(approval.matched_policies) && approval.matched_policies.length > 0
            ? approval.matched_policies.join(', ')
            : '-'}
        </dd>

        <dt>Request Payload</dt>
        <dd>
          <pre style={{ whiteSpace: 'pre-wrap', fontSize: '0.8rem' }}>
            {JSON.stringify(payload, null, 2)}
          </pre>
        </dd>
      </dl>

      {isPending && (
        <div style={{ marginTop: '1.5rem' }}>
          <label htmlFor="comment" style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 500 }}>
            Review Comment
          </label>
          <textarea
            id="comment"
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            rows={3}
            style={{ width: '100%', marginBottom: '1rem' }}
            placeholder="Optional comment..."
          />
          <div className="actions">
            <button
              className="btn btn-primary"
              onClick={() => handleReview('approved')}
              disabled={submitting}
            >
              {submitting ? 'Submitting...' : 'Approve'}
            </button>
            <button
              className="btn btn-danger"
              onClick={() => handleReview('denied')}
              disabled={submitting}
            >
              {submitting ? 'Submitting...' : 'Deny'}
            </button>
          </div>
        </div>
      )}

      <div className="actions" style={{ marginTop: '1rem' }}>
        <button className="btn btn-secondary" onClick={() => navigate('/approvals')}>
          Back to Approvals
        </button>
      </div>
    </>
  );
}
