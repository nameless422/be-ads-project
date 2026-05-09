import { Activity, BarChart3, Gauge, Layers3, Palette, SlidersHorizontal } from "lucide-react";
import type { ReactNode } from "react";
import type { FilterState, PageKey } from "../api/types";
import { pageMeta } from "../utils/routing";
import { Filters } from "./Filters";

type AppShellProps = {
  page: PageKey;
  filters: FilterState;
  loading: boolean;
  error: string | null;
  generatedAt?: string;
  children: ReactNode;
  onNavigate: (page: PageKey) => void;
  onApplyFilters: (filters: FilterState) => void;
};

const navItems: Array<{ key: PageKey; icon: ReactNode }> = [
  { key: "overview", icon: <Gauge size={17} /> },
  { key: "breakdown", icon: <Layers3 size={17} /> },
  { key: "creatives", icon: <Palette size={17} /> },
  { key: "quality", icon: <Activity size={17} /> },
  { key: "control", icon: <SlidersHorizontal size={17} /> },
];

export function AppShell({ page, filters, loading, error, generatedAt, children, onNavigate, onApplyFilters }: AppShellProps) {
  const meta = pageMeta[page];

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand">
          <div className="brand-mark">
            <BarChart3 size={20} />
          </div>
          <div>
            <strong>be_ads BI</strong>
            <span>海外游戏 UA 工作台</span>
          </div>
        </div>
        <nav className="tabs" aria-label="BI sections">
          {navItems.map((item) => (
            <button key={item.key} type="button" className={item.key === page ? "active" : ""} onClick={() => onNavigate(item.key)}>
              {item.icon}
              {pageMeta[item.key].title}
            </button>
          ))}
        </nav>
        <a className="legacy-link" href="/">
          Control Panel
        </a>
      </header>

      <main className="workspace">
        <section className="page-head">
          <div>
            <h1>{meta.title}</h1>
            <p>{meta.description}</p>
          </div>
          <div className="sync-state">
            <span className={loading ? "dot loading" : "dot"} />
            <strong>{loading ? "Loading" : "Synced"}</strong>
            <small>{generatedAt ? generatedAt.slice(0, 19).replace("T", " ") : "live API"}</small>
          </div>
        </section>

        <Filters filters={filters} onApply={onApplyFilters} />
        {error ? <div className="error-banner">{error}</div> : null}
        {children}
      </main>
    </div>
  );
}
