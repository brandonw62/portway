import { useState } from 'react';
import { createTeam, createProject } from '../api';
import { useWorkspace } from '../WorkspaceContext';

export default function TeamsPage() {
  const { teams, projects, activeTeam, activeProject } = useWorkspace();

  const [teamName, setTeamName] = useState('');
  const [projectName, setProjectName] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const handleCreateTeam = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    if (!teamName.trim()) return;
    try {
      await createTeam({ name: teamName.trim() });
      setTeamName('');
      setSuccess('Team created. Reload the page to see it in the switcher.');
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const handleCreateProject = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    if (!projectName.trim() || !activeTeam) return;
    try {
      await createProject(activeTeam.id, { name: projectName.trim() });
      setProjectName('');
      setSuccess('Project created. Reload the page to see it in the switcher.');
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <>
      <h2>Teams &amp; Projects</h2>
      {error && <div className="error">{error}</div>}
      {success && <div className="success">{success}</div>}

      <section style={{ marginBottom: '2rem' }}>
        <h3>Your Teams</h3>
        {teams.length === 0 ? (
          <p className="empty">You don't belong to any teams yet.</p>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Slug</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              {teams.map((t) => (
                <tr key={t.id} className={t.id === activeTeam?.id ? 'active-row' : ''}>
                  <td>{t.name}</td>
                  <td style={{ color: 'var(--text-muted)' }}>{t.slug}</td>
                  <td style={{ color: 'var(--text-muted)' }}>{t.description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        <form onSubmit={handleCreateTeam} style={{ display: 'flex', gap: '0.5rem', marginTop: '1rem' }}>
          <input
            type="text"
            placeholder="New team name"
            value={teamName}
            onChange={(e) => setTeamName(e.target.value)}
          />
          <button className="btn btn-primary" type="submit">
            Create Team
          </button>
        </form>
      </section>

      <section>
        <h3>Projects in {activeTeam?.name ?? '...'}</h3>
        {projects.length === 0 ? (
          <p className="empty">No projects in this team yet.</p>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Slug</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              {projects.map((p) => (
                <tr key={p.id} className={p.id === activeProject?.id ? 'active-row' : ''}>
                  <td>{p.name}</td>
                  <td style={{ color: 'var(--text-muted)' }}>{p.slug}</td>
                  <td style={{ color: 'var(--text-muted)' }}>{p.description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        <form
          onSubmit={handleCreateProject}
          style={{ display: 'flex', gap: '0.5rem', marginTop: '1rem' }}
        >
          <input
            type="text"
            placeholder="New project name"
            value={projectName}
            onChange={(e) => setProjectName(e.target.value)}
            disabled={!activeTeam}
          />
          <button className="btn btn-primary" type="submit" disabled={!activeTeam}>
            Create Project
          </button>
        </form>
      </section>
    </>
  );
}
