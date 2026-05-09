import { useCallback, useEffect, useMemo, useState } from "react";
import { loadBIDashboard } from "./api/client";
import type { BIDashboardData, FilterState, PageKey } from "./api/types";
import { AppShell } from "./components/AppShell";
import { BreakdownPage } from "./pages/BreakdownPage";
import { ControlPage } from "./pages/ControlPage";
import { CreativesPage } from "./pages/CreativesPage";
import { OverviewPage } from "./pages/OverviewPage";
import { QualityPage } from "./pages/QualityPage";
import { filtersToSearch, pathForPage, readFiltersFromSearch, readPageFromPath } from "./utils/routing";

type LocationState = {
  page: PageKey;
  filters: FilterState;
};

function readLocation(): LocationState {
  return {
    page: readPageFromPath(window.location.pathname),
    filters: readFiltersFromSearch(window.location.search),
  };
}

function App() {
  const [location, setLocation] = useState<LocationState>(() => readLocation());
  const [data, setData] = useState<BIDashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const filterKey = useMemo(() => filtersToSearch(location.filters), [location.filters]);

  useEffect(() => {
    function handlePopState() {
      setLocation(readLocation());
    }
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError(null);
    loadBIDashboard(location.filters)
      .then((result) => {
        if (!active) {
          return;
        }
        setData(result);
      })
      .catch((err) => {
        if (!active) {
          return;
        }
        setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [filterKey, reloadToken, location.filters]);

  const navigate = useCallback(
    (page: PageKey) => {
      const target = pathForPage(page, location.filters);
      window.history.pushState({}, "", target);
      setLocation(readLocation());
    },
    [location.filters],
  );

  const applyFilters = useCallback(
    (filters: FilterState) => {
      const target = pathForPage(location.page, filters);
      window.history.pushState({}, "", target);
      setLocation(readLocation());
    },
    [location.page],
  );

  const refresh = useCallback(() => {
    setReloadToken((current) => current + 1);
  }, []);

  const generatedAt = data?.controlOverview?.GeneratedAt;

  return (
    <AppShell
      page={location.page}
      filters={location.filters}
      loading={loading}
      error={error}
      generatedAt={generatedAt}
      onNavigate={navigate}
      onApplyFilters={applyFilters}
    >
      {data ? (
        <>
          {location.page === "overview" ? <OverviewPage data={data} /> : null}
          {location.page === "breakdown" ? <BreakdownPage data={data} /> : null}
          {location.page === "creatives" ? <CreativesPage data={data} /> : null}
          {location.page === "quality" ? <QualityPage data={data} /> : null}
          {location.page === "control" ? <ControlPage data={data} onRefresh={refresh} /> : null}
        </>
      ) : (
        <div className="panel loading-panel">Loading BI workspace...</div>
      )}
    </AppShell>
  );
}

export default App;
