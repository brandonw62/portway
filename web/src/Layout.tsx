import { NavLink, Outlet } from 'react-router-dom';

export default function Layout() {
  return (
    <div className="layout">
      <nav className="sidebar">
        <h1>Portway</h1>
        <NavLink to="/" className={({ isActive }) => isActive ? 'active' : ''} end>
          Resource Catalog
        </NavLink>
        <NavLink to="/resources" className={({ isActive }) => isActive ? 'active' : ''}>
          My Resources
        </NavLink>
        <NavLink to="/resources/new" className={({ isActive }) => isActive ? 'active' : ''}>
          Provision New
        </NavLink>
        <NavLink to="/approvals" className={({ isActive }) => isActive ? 'active' : ''}>
          Approvals
        </NavLink>
      </nav>
      <main className="main">
        <Outlet />
      </main>
    </div>
  );
}
