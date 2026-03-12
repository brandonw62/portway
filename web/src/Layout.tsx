import { NavLink, Outlet } from 'react-router-dom';
import { useWorkspace } from './WorkspaceContext';

export default function Layout() {
  const { teams, projects, activeTeam, activeProject, setActiveTeam, setActiveProject, loading } =
    useWorkspace();

  return (
    <div className="layout">
      <nav className="sidebar">
        <h1>Portway</h1>

        <div className="workspace-switcher">
          <label>Team</label>
          <select
            value={activeTeam?.id ?? ''}
            onChange={(e) => {
              const t = teams.find((t) => t.id === e.target.value);
              if (t) setActiveTeam(t);
            }}
            disabled={loading}
          >
            {teams.length === 0 && <option value="">No teams</option>}
            {teams.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>

          <label>Project</label>
          <select
            value={activeProject?.id ?? ''}
            onChange={(e) => {
              const p = projects.find((p) => p.id === e.target.value);
              if (p) setActiveProject(p);
            }}
            disabled={loading || !activeTeam}
          >
            {projects.length === 0 && <option value="">No projects</option>}
            {projects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </div>

        <NavLink to="/" className={({ isActive }) => (isActive ? 'active' : '')} end>
          Resource Catalog
        </NavLink>
        <NavLink to="/resources" className={({ isActive }) => (isActive ? 'active' : '')}>
          My Resources
        </NavLink>
        <NavLink to="/resources/new" className={({ isActive }) => (isActive ? 'active' : '')}>
          Provision New
        </NavLink>
        <NavLink to="/approvals" className={({ isActive }) => (isActive ? 'active' : '')}>
          Approvals
        </NavLink>
        <NavLink to="/teams" className={({ isActive }) => (isActive ? 'active' : '')}>
          Teams
        </NavLink>
      </nav>
      <main className="main">
        <Outlet />
      </main>
    </div>
  );
}
