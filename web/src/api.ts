const BASE = '/api/v1';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'X-User-Id': 'dev-user-001', // placeholder until auth
      ...init?.headers,
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
  return res.json();
}

export interface ResourceType {
  id: string;
  name: string;
  slug: string;
  category: string;
  description: string;
  default_spec: Record<string, unknown>;
  spec_schema: Record<string, unknown>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface Resource {
  id: string;
  project_id: string;
  resource_type_id: string;
  name: string;
  slug: string;
  status: string;
  spec: Record<string, unknown>;
  provider_ref: string;
  status_message: string;
  requested_by: string;
  created_at: string;
  updated_at: string;
}

export function listResourceTypes(): Promise<ResourceType[]> {
  return request('/resource-types');
}

export function getResourceType(id: string): Promise<ResourceType> {
  return request(`/resource-types/${id}`);
}

export function listResources(projectId: string): Promise<Resource[]> {
  return request(`/resources?project_id=${encodeURIComponent(projectId)}`);
}

export function getResource(id: string): Promise<Resource> {
  return request(`/resources/${id}`);
}

export function createResource(body: {
  project_id: string;
  resource_type_id: string;
  name: string;
  spec?: Record<string, unknown>;
}): Promise<Resource> {
  return request('/resources', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function deleteResource(id: string): Promise<Resource> {
  return request(`/resources/${id}`, { method: 'DELETE' });
}

export interface ApprovalRequest {
  id: string;
  project_id: string;
  requested_by: string;
  resource_type: string;
  request_payload: Record<string, unknown>;
  reasons: string[];
  matched_policies: string[];
  status: string;
  reviewed_by: string | null;
  review_comment: string;
  reviewed_at: string | null;
  expires_at: string;
  created_at: string;
  updated_at: string;
}

export function listApprovals(): Promise<ApprovalRequest[]> {
  return request('/approvals');
}

export function getApproval(id: string): Promise<ApprovalRequest> {
  return request(`/approvals/${id}`);
}

export function reviewApproval(
  id: string,
  body: { decision: 'approved' | 'denied'; comment: string },
): Promise<ApprovalRequest> {
  return request(`/approvals/${id}/review`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

// --- Teams ---

export interface Team {
  id: string;
  name: string;
  slug: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export function listTeams(): Promise<Team[]> {
  return request('/teams');
}

export function getTeam(id: string): Promise<Team> {
  return request(`/teams/${id}`);
}

export function createTeam(body: { name: string; description?: string }): Promise<Team> {
  return request('/teams', { method: 'POST', body: JSON.stringify(body) });
}

// --- Projects ---

export interface Project {
  id: string;
  team_id: string;
  name: string;
  slug: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export function listProjects(teamId: string): Promise<Project[]> {
  return request(`/teams/${teamId}/projects`);
}

export function getProject(teamId: string, projectId: string): Promise<Project> {
  return request(`/teams/${teamId}/projects/${projectId}`);
}

export function createProject(
  teamId: string,
  body: { name: string; description?: string },
): Promise<Project> {
  return request(`/teams/${teamId}/projects`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

// --- Auth ---

export interface AuthUser {
  id: string;
  email: string;
  name: string;
}

export function getMe(): Promise<AuthUser> {
  return request('/auth/me');
}
