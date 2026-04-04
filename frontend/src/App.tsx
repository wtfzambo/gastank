import { useState, useEffect, useCallback } from 'react';
import './App.css';
import {
  GetAuthStatus,
  StartGitHubLogin,
  PollGitHubLogin,
  LogOut,
  GetCopilotUsage,
} from '../wailsjs/go/main/App';

// ---- Type definitions (mirroring Go structs) ----

interface AuthStatus {
  authenticated: boolean;
  source?: string;
}

interface DeviceFlowState {
  deviceCode: string;
  userCode: string;
  verificationURI: string;
  expiresIn: number;
  interval: number;
}

interface UsageReport {
  provider: string;
  retrievedAt: string;
  metrics: Record<string, number>;
  metadata?: Record<string, string>;
}

// ---- Helpers ----

function pct(value: number | undefined): string {
  if (value === undefined) return '—';
  return `${Math.round(value)}%`;
}

function MetricRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="metric-row">
      <span className="metric-label">{label}</span>
      <span className="metric-value">{value}</span>
    </div>
  );
}

// ---- Login screen ----

function LoginScreen({ onLoggedIn }: { onLoggedIn: () => void }) {
  const [flow, setFlow] = useState<DeviceFlowState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);
  const [polling, setPolling] = useState(false);

  async function startLogin() {
    setStarting(true);
    setError(null);
    try {
      const state = await StartGitHubLogin();
      setFlow(state);
      beginPolling(state);
    } catch (e: unknown) {
      setError(String(e));
    } finally {
      setStarting(false);
    }
  }

  function beginPolling(state: DeviceFlowState) {
    setPolling(true);
    // interval from GitHub + a small buffer (ms)
    const intervalMs = Math.max((state.interval + 1) * 1000, 6000);

    const timer = setInterval(async () => {
      try {
        const done = await PollGitHubLogin(state.deviceCode);
        if (done) {
          clearInterval(timer);
          setPolling(false);
          onLoggedIn();
        }
      } catch (e: unknown) {
        clearInterval(timer);
        setPolling(false);
        setFlow(null);
        setError(String(e));
      }
    }, intervalMs);
  }

  return (
    <div className="login-screen">
      <h2 className="login-title">Connect GitHub Copilot</h2>
      <p className="login-body">
        Sign in with your GitHub account to view Copilot usage.
      </p>

      {!flow && (
        <button className="primary-btn" onClick={startLogin} disabled={starting}>
          {starting ? 'Starting…' : 'Sign in with GitHub'}
        </button>
      )}

      {flow && (
        <div className="device-code-box">
          <p className="device-instruction">
            Open{' '}
            <a href={flow.verificationURI} target="_blank" rel="noreferrer">
              {flow.verificationURI}
            </a>{' '}
            and enter this code:
          </p>
          <div className="device-code">{flow.userCode}</div>
          {polling && <p className="waiting-text">Waiting for approval…</p>}
        </div>
      )}

      {error && <p className="login-error">{error}</p>}
    </div>
  );
}

// ---- Usage screen ----

function UsageScreen({ onLogOut }: { onLogOut: () => void }) {
  const [report, setReport] = useState<UsageReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const r = await GetCopilotUsage();
      setReport(r);
    } catch (e: unknown) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { refresh(); }, [refresh]);

  const m = report?.metrics ?? {};
  const meta = report?.metadata ?? {};

  return (
    <>
      <main className="usage-card">
        {loading && <p className="status-text">Loading…</p>}

        {!loading && error && (
          <div className="error-box">
            <p className="error-title">Could not fetch usage</p>
            <p className="error-detail">{error}</p>
          </div>
        )}

        {!loading && report && !error && (
          <>
            <div className="plan-row">
              <span className="plan-badge">{meta.plan ?? 'GitHub Copilot'}</span>
              {meta.quota_reset_date && (
                <span className="reset-date">Resets {meta.quota_reset_date}</span>
              )}
            </div>

            <div className="metrics">
              <MetricRow
                label="Premium interactions remaining"
                value={m['premium_unlimited'] === 1 ? 'Unlimited' : pct(m['premium_percent_remaining'])}
              />
              <MetricRow
                label="Chat remaining"
                value={m['chat_unlimited'] === 1 ? 'Unlimited' : pct(m['chat_percent_remaining'])}
              />
              <MetricRow
                label="Completions remaining"
                value={m['completions_unlimited'] === 1 ? 'Unlimited' : pct(m['completions_percent_remaining'])}
              />
            </div>

            <p className="retrieved-at">
              Updated {new Date(report.retrievedAt).toLocaleTimeString()}
            </p>
          </>
        )}
      </main>

      <div className="action-row">
        <button className="refresh-btn" onClick={refresh} disabled={loading}>
          {loading ? 'Refreshing…' : 'Refresh'}
        </button>
        <button className="logout-btn" onClick={onLogOut}>Log out</button>
      </div>
    </>
  );
}

// ---- Root component ----

type Screen = 'loading' | 'login' | 'usage';

function App() {
  const [screen, setScreen] = useState<Screen>('loading');

  useEffect(() => {
    GetAuthStatus().then((status: AuthStatus) => {
      setScreen(status.authenticated ? 'usage' : 'login');
    });
  }, []);

  function handleLogOut() {
    LogOut();
    setScreen('login');
  }

  return (
    <div id="App">
      <header className="app-header">
        <h1>ingo</h1>
        <p className="app-subtitle">AI token usage monitor</p>
      </header>

      {screen === 'loading' && <p className="status-text">Loading…</p>}
      {screen === 'login' && <LoginScreen onLoggedIn={() => setScreen('usage')} />}
      {screen === 'usage' && <UsageScreen onLogOut={handleLogOut} />}
    </div>
  );
}

export default App;
