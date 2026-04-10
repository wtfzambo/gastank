import { useState, useEffect, useCallback, useRef } from 'react';
import { Browser } from '@wailsio/runtime';
import './App.css';
import {
  GetAuthStatus,
  StartGitHubLogin,
  PollGitHubLogin,
  LogOut,
  GetCopilotUsage,
} from '../bindings/gastank/app';
import { AuthStatus, DeviceFlowState } from '../bindings/gastank/models';
import { UsageReport } from '../bindings/gastank/internal/usage/models';

// ---- Helpers ----

function pct(value: number | undefined): string {
  if (value === undefined) return '—';
  return `${Math.round(value)}%`;
}

function isAuthError(msg: string): boolean {
  return /401|403|unauthorized|forbidden|token.*invalid|invalid.*token|not authenticated|log.*in/i.test(msg);
}

function MetricRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="metric-row">
      <span className="metric-label">{label}</span>
      <span className="metric-value">{value}</span>
    </div>
  );
}

function CopyCode({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div className="device-code-row">
      <span className="device-code">{code}</span>
      <button
        className="copy-btn"
        onClick={handleCopy}
        title="Copy code"
        aria-label="Copy code to clipboard"
      >
        {copied ? (
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        ) : (
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
          </svg>
        )}
      </button>
    </div>
  );
}

// ---- Login screen ----

function LoginScreen({
  onLoggedIn,
  sessionExpired,
}: {
  onLoggedIn: () => void;
  sessionExpired: boolean;
}) {
  const [flow, setFlow] = useState<DeviceFlowState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);
  const [polling, setPolling] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Clean up polling timer on unmount.
  useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        clearInterval(timerRef.current);
      }
    };
  }, []);

  async function startLogin() {
    setStarting(true);
    setError(null);
    try {
      const state = await StartGitHubLogin();
      if (!state) {
        setError('Failed to start login flow');
        return;
      }
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

    timerRef.current = setInterval(async () => {
      try {
        const done = await PollGitHubLogin(state.deviceCode);
        if (done) {
          if (timerRef.current !== null) clearInterval(timerRef.current);
          timerRef.current = null;
          setPolling(false);
          onLoggedIn();
        }
      } catch (e: unknown) {
        if (timerRef.current !== null) clearInterval(timerRef.current);
        timerRef.current = null;
        setPolling(false);
        setFlow(null);
        setError(String(e));
      }
    }, intervalMs);
  }

  return (
    <div className="login-screen">
      <h2 className="login-title">Connect GitHub Copilot</h2>
      {sessionExpired && (
        <p className="session-expired">Session expired — please sign in again.</p>
      )}
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
            <a
              href="#"
              className="verification-link"
              onClick={(e) => {
                e.preventDefault();
                Browser.OpenURL(flow.verificationURI);
              }}
            >
              {flow.verificationURI}
            </a>{' '}
            and enter this code:
          </p>
          <CopyCode code={flow.userCode} />
          {polling && <p className="waiting-text">Waiting for approval…</p>}
        </div>
      )}

      {error && <p className="login-error">{error}</p>}
    </div>
  );
}

// ---- Usage screen ----

function UsageScreen({
  onLogOut,
  onAuthError,
}: {
  onLogOut: () => void;
  onAuthError: () => void;
}) {
  const [report, setReport] = useState<UsageReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [showDetails, setShowDetails] = useState(false);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const r = await GetCopilotUsage();
      setReport(r);
    } catch (e: unknown) {
      const msg = String(e);
      if (isAuthError(msg)) {
        // Token invalid/expired — clear and send back to login.
        await LogOut();
        onAuthError();
        return;
      }
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [onAuthError]);

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
            </div>

            <button
              type="button"
              className="details-toggle"
              onClick={() => setShowDetails(prev => !prev)}
              aria-expanded={showDetails}
              aria-controls="details-metrics"
            >
              {showDetails ? 'Hide details ▴' : 'Show details ▾'}
            </button>

            {showDetails && (
              <div id="details-metrics" className="metrics details-metrics">
                <MetricRow
                  label="Chat remaining"
                  value={m['chat_unlimited'] === 1 ? 'Unlimited' : pct(m['chat_percent_remaining'])}
                />
                <MetricRow
                  label="Completions remaining"
                  value={m['completions_unlimited'] === 1 ? 'Unlimited' : pct(m['completions_percent_remaining'])}
                />
              </div>
            )}

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
  const [sessionExpired, setSessionExpired] = useState(false);

  useEffect(() => {
    GetAuthStatus().then((status: AuthStatus) => {
      setScreen(status.authenticated ? 'usage' : 'login');
    });
  }, []);

  function handleLogOut() {
    LogOut();
    setSessionExpired(false);
    setScreen('login');
  }

  function handleAuthError() {
    setSessionExpired(true);
    setScreen('login');
  }

  return (
    <div id="App">
      <header className="app-header">
        <h1>gastank</h1>
        <p className="app-subtitle">AI token usage monitor</p>
      </header>

      {screen === 'loading' && <p className="status-text">Loading…</p>}
      {screen === 'login' && (
        <LoginScreen
          onLoggedIn={() => { setSessionExpired(false); setScreen('usage'); }}
          sessionExpired={sessionExpired}
        />
      )}
      {screen === 'usage' && (
        <UsageScreen onLogOut={handleLogOut} onAuthError={handleAuthError} />
      )}
    </div>
  );
}

export default App;
