import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { WorkspaceProvider } from './WorkspaceContext';
import Layout from './Layout';
import CatalogPage from './pages/CatalogPage';
import ResourcesPage from './pages/ResourcesPage';
import ResourceDetailPage from './pages/ResourceDetailPage';
import ProvisionPage from './pages/ProvisionPage';
import ApprovalListPage from './pages/ApprovalListPage';
import ApprovalDetailPage from './pages/ApprovalDetailPage';
import TeamsPage from './pages/TeamsPage';

export default function App() {
  return (
    <BrowserRouter>
      <WorkspaceProvider>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<CatalogPage />} />
            <Route path="resources" element={<ResourcesPage />} />
            <Route path="resources/new" element={<ProvisionPage />} />
            <Route path="resources/:id" element={<ResourceDetailPage />} />
            <Route path="approvals" element={<ApprovalListPage />} />
            <Route path="approvals/:id" element={<ApprovalDetailPage />} />
            <Route path="teams" element={<TeamsPage />} />
          </Route>
        </Routes>
      </WorkspaceProvider>
    </BrowserRouter>
  );
}
