import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { Navigate, Outlet, useLocation } from "react-router-dom";

export interface Employee {
  id: number;
  name: string;
  email: string;
}

interface AuthSession {
  token: string;
  employee: Employee;
}

interface AuthContextValue {
  token: string | null;
  employee: Employee | null;
  isAuthenticated: boolean;
  signIn: (token: string, employee: Employee) => void;
  signOut: () => void;
}

const STORAGE_KEY = "werkstattportal.session";

const AuthContext = createContext<AuthContextValue | null>(null);

function readStoredSession(): AuthSession | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<AuthSession> | null;
    if (
      parsed &&
      typeof parsed.token === "string" &&
      parsed.employee &&
      typeof parsed.employee.name === "string" &&
      typeof parsed.employee.email === "string"
    ) {
      return { token: parsed.token, employee: parsed.employee };
    }
  } catch {
    // Ignore malformed or unavailable storage — the user simply starts signed out.
  }
  return null;
}

function writeStoredSession(session: AuthSession | null): void {
  try {
    if (session) {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // Storage may be unavailable (private mode); the in-memory session still works.
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  // The session is restored from storage so it survives a page reload.
  const [session, setSession] = useState<AuthSession | null>(() => readStoredSession());

  const signIn = useCallback((token: string, employee: Employee) => {
    const next: AuthSession = { token, employee };
    setSession(next);
    writeStoredSession(next);
  }, []);

  const signOut = useCallback(() => {
    setSession(null);
    writeStoredSession(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      token: session?.token ?? null,
      employee: session?.employee ?? null,
      isAuthenticated: session !== null,
      signIn,
      signOut,
    }),
    [session, signIn, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}

/**
 * Layout route guarding the workshop area. Without a held token the visitor is
 * redirected to the workshop login (which is inert until its own ticket).
 */
export function ProtectedRoute() {
  const { isAuthenticated } = useAuth();
  const location = useLocation();

  if (!isAuthenticated) {
    return (
      <Navigate
        to="/werkstatt/login"
        replace
        state={{ from: location.pathname }}
      />
    );
  }

  return <Outlet />;
}
