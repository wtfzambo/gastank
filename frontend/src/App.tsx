import { Browser } from '@wailsio/runtime';
import { useCallback, useEffect, useRef, useState } from 'react';
import {
  GetAuthStatus,
  GetCopilotUsage,
  LogOut,
  PollGitHubLogin,
  StartGitHubLogin,
} from '../bindings/gastank/app';
import { UsageReport } from '../bindings/gastank/internal/usage/models';
import { AuthStatus, DeviceFlowState } from '../bindings/gastank/models';
import './App.css';

// ---- Helpers ----

const AUTO_REFRESH_INTERVAL_MS = 5 * 60 * 1000;

function pct(value: number | undefined): string {
  if (value === undefined) return '—';
  return `${value}%`;
}

function formatRetrievedAt(value: string | undefined): string {
  if (!value) return '—';

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';

  return date.toLocaleTimeString([], {
    hour: 'numeric',
    minute: '2-digit',
  });
}

function quotaValue(metrics: UsageReport['metrics'] | undefined, prefix: string): string {
  if (!metrics) return '—';
  if (metrics[`${prefix}_unlimited`] === 1) return 'Unlimited';
  return pct(metrics[`${prefix}_percent_remaining`]);
}

function isAuthError(msg: string): boolean {
  return /401|403|unauthorized|forbidden|token.*invalid|invalid.*token|not authenticated|log.*in/i.test(msg);
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
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        ) : (
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
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
    <main className="login-screen">
      <div className="login-header">
        <h2>Connect GitHub Copilot</h2>
        <p>Sign in to keep track of your Copilot usage.</p>
      </div>

      {sessionExpired && (
        <div className="error-text">
          <p>Session expired</p>
          <span>Sign in again to continue tracking usage.</span>
        </div>
      )}

      {!flow && (
        <button className="btn btn-outline" onClick={startLogin} disabled={starting}>
          {starting ? 'Starting…' : 'Sign in with GitHub'}
        </button>
      )}

      {flow && (
        <div className="login-flow-container">
          <div className="login-instructions">
            <span>1. Open</span>
            <a
              href="#"
              className="login-link"
              onClick={(e) => {
                e.preventDefault();
                Browser.OpenURL(flow.verificationURI);
              }}
            >
              {flow.verificationURI}
            </a>
          </div>
          <div className="login-instructions">
            <span>2. Enter this code:</span>
          </div>

          <CopyCode code={flow.userCode} />

          {polling && (
            <div className="login-polling">
              <svg className="spinning" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="23 4 23 10 17 10"></polyline>
                <polyline points="1 20 1 14 7 14"></polyline>
                <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
              </svg>
              <span>Waiting for approval…</span>
            </div>
          )}
        </div>
      )}

      {error && (
        <div className="error-text">
          <p>Login failed</p>
          <span>{error}</span>
        </div>
      )}
    </main>
  );
}

// ---- Usage screen ----

function UsageScreen({
  refreshSignal,
  onRefreshingChange,
  onAuthError,
}: {
  refreshSignal: number;
  onRefreshingChange: (isRefreshing: boolean) => void;
  onAuthError: () => void;
}) {
  const [report, setReport] = useState<UsageReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showDetails, setShowDetails] = useState(false);
  const refreshInFlightRef = useRef(false);

  const refresh = useCallback(async () => {
    if (refreshInFlightRef.current) {
      return;
    }

    refreshInFlightRef.current = true;
    onRefreshingChange(true);
    setError(null);

    try {
      const r = await GetCopilotUsage();
      setReport(r);
    } catch (e: unknown) {
      const msg = String(e);
      if (isAuthError(msg)) {
        await LogOut();
        onAuthError();
        return;
      }
      setError(msg);
    } finally {
      refreshInFlightRef.current = false;
      onRefreshingChange(false);
    }
  }, [onAuthError, onRefreshingChange]);

  useEffect(() => {
    void refresh();
  }, [refresh, refreshSignal]);

  const m = report?.metrics ?? {};
  const meta = report?.metadata ?? {};
  const hasReport = report !== null;
  const hasStaleData = hasReport && error !== null;
  const showEmptyError = error !== null && !hasReport;

  const primaryValue = quotaValue(m, 'premium');
  const chatValue = quotaValue(m, 'chat');
  const completionsValue = quotaValue(m, 'completions');

  return (
    <main className="usage-screen">
      {!hasReport && !error && <p style={{opacity: 0.6}}>Loading…</p>}

      {showEmptyError && (
        <div className="error-text">
          <p>Could not fetch usage</p>
          <span>{error}</span>
        </div>
      )}

      {hasReport && (
        <div className="provider-list">
          <div className="provider-item">
            {hasStaleData && (
              <div className="error-text">
                <p>Last refresh failed</p>
                <span>Showing the most recent cached snapshot.</span>
              </div>
            )}

            <div className="provider-header">
              <h2 className="provider-name">GitHub Copilot</h2>
              <span className="provider-badge">{meta.plan ?? 'Subscription'}</span>
            </div>

            <div className="primary-metric">
              <span className="primary-metric-label">Premium interactions left</span>
              <span className="primary-metric-value">{primaryValue}</span>
            </div>

            <button
              className={`details-toggle ${showDetails ? 'open' : ''}`}
              onClick={() => setShowDetails(!showDetails)}
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="9 18 15 12 9 6" />
              </svg>
              <span>{showDetails ? 'Hide details' : 'Show details'}</span>
            </button>

            {showDetails && (
              <div className="details-section">
                <div className="metric-row">
                  <span className="metric-label">Chat</span>
                  <span className="metric-value">{chatValue}</span>
                </div>
                <div className="metric-row">
                  <span className="metric-label">Completions</span>
                  <span className="metric-value">{completionsValue}</span>
                </div>
              </div>
            )}

            <div className="provider-footer">
              <div className="meta-info">
                <span>Refresh: {formatRetrievedAt(report.retrievedAt)}</span>
                {meta.quota_reset_date && (
                  <>
                    <span className="meta-separator">•</span>
                    <span>Reset: {meta.quota_reset_date}</span>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}

// ---- Root component ----

type Screen = 'loading' | 'login' | 'usage';

function App() {
  const [screen, setScreen] = useState<Screen>('loading');
  const [sessionExpired, setSessionExpired] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [refreshSignal, setRefreshSignal] = useState(0);

  useEffect(() => {
    GetAuthStatus().then((status: AuthStatus) => {
      setScreen(status.authenticated ? 'usage' : 'login');
    });
  }, []);

  useEffect(() => {
    const interval = window.setInterval(() => {
      if (screen === 'usage') {
        triggerRefresh();
      }
    }, AUTO_REFRESH_INTERVAL_MS);
    return () => window.clearInterval(interval);
  }, [screen]);

  const handleLogOut = useCallback(() => {
    LogOut();
    setSessionExpired(false);
    setScreen('login');
  }, []);

  const handleAuthError = useCallback(() => {
    setSessionExpired(true);
    setScreen('login');
  }, []);

  const triggerRefresh = useCallback(() => {
    setRefreshSignal((prev) => prev + 1);
  }, []);

  return (
    <div id="App">
      <header className="app-header">
        <div className="header-brand">
          <div className="brand-mark" aria-hidden="true" />
          <div>
            <h1>gastank</h1>
            <p className="app-subtitle">AI usage monitor</p>
          </div>
        </div>
      </header>

      <div className="screen-container">
        {screen === 'loading' && <p style={{opacity: 0.6}}>Loading…</p>}

        {screen === 'login' && (
          <LoginScreen
            onLoggedIn={() => { setSessionExpired(false); setScreen('usage'); }}
            sessionExpired={sessionExpired}
          />
        )}

        {screen === 'usage' && (
          <UsageScreen
            refreshSignal={refreshSignal}
            onRefreshingChange={setIsRefreshing}
            onAuthError={handleAuthError}
          />
        )}
      </div>

      {screen === 'usage' && (
        <footer className="app-footer">
          <button
            className="footer-btn refresh-btn"
            onClick={triggerRefresh}
            disabled={isRefreshing}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={isRefreshing ? "spinning" : ""}>
              <polyline points="23 4 23 10 17 10"></polyline>
              <polyline points="1 20 1 14 7 14"></polyline>
              <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
            </svg>
            Refresh all
          </button>
          <button
            className="footer-btn"
            onClick={handleLogOut}
          >
            Log out
          </button>
        </footer>
      )}
    </div>
  );
}

export default App;
