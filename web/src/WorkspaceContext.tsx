import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { listTeams, listProjects, type Team, type Project } from './api';

interface WorkspaceState {
  teams: Team[];
  projects: Project[];
  activeTeam: Team | null;
  activeProject: Project | null;
  setActiveTeam: (team: Team) => void;
  setActiveProject: (project: Project) => void;
  loading: boolean;
}

const WorkspaceContext = createContext<WorkspaceState | null>(null);

export function useWorkspace(): WorkspaceState {
  const ctx = useContext(WorkspaceContext);
  if (!ctx) throw new Error('useWorkspace must be used within WorkspaceProvider');
  return ctx;
}

export function WorkspaceProvider({ children }: { children: ReactNode }) {
  const [teams, setTeams] = useState<Team[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [activeTeam, setActiveTeamState] = useState<Team | null>(null);
  const [activeProject, setActiveProjectState] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);

  // Load teams on mount.
  useEffect(() => {
    listTeams()
      .then((t) => {
        setTeams(t);
        // Auto-select first team, or restore from localStorage.
        const savedTeamId = localStorage.getItem('portway:activeTeamId');
        const saved = t.find((team) => team.id === savedTeamId);
        if (saved) {
          setActiveTeamState(saved);
        } else if (t.length > 0) {
          setActiveTeamState(t[0]);
        } else {
          setLoading(false);
        }
      })
      .catch(() => setLoading(false));
  }, []);

  // Load projects when active team changes.
  useEffect(() => {
    if (!activeTeam) {
      setProjects([]);
      return;
    }
    listProjects(activeTeam.id)
      .then((p) => {
        setProjects(p);
        const savedProjectId = localStorage.getItem('portway:activeProjectId');
        const saved = p.find((proj) => proj.id === savedProjectId);
        if (saved) {
          setActiveProjectState(saved);
        } else if (p.length > 0) {
          setActiveProjectState(p[0]);
        }
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [activeTeam]);

  const setActiveTeam = (team: Team) => {
    setActiveTeamState(team);
    setActiveProjectState(null);
    localStorage.setItem('portway:activeTeamId', team.id);
    localStorage.removeItem('portway:activeProjectId');
  };

  const setActiveProject = (project: Project) => {
    setActiveProjectState(project);
    localStorage.setItem('portway:activeProjectId', project.id);
  };

  return (
    <WorkspaceContext.Provider
      value={{
        teams,
        projects,
        activeTeam,
        activeProject,
        setActiveTeam,
        setActiveProject,
        loading,
      }}
    >
      {children}
    </WorkspaceContext.Provider>
  );
}
