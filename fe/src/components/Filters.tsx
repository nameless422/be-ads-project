import { RotateCcw, Search } from "lucide-react";
import { useEffect, useState } from "react";
import type { FilterState } from "../api/types";
import { defaultFilters } from "../utils/routing";

type FiltersProps = {
  filters: FilterState;
  onApply: (filters: FilterState) => void;
};

export function Filters({ filters, onApply }: FiltersProps) {
  const [draft, setDraft] = useState(filters);

  useEffect(() => {
    setDraft(filters);
  }, [filters]);

  function update(field: keyof FilterState, value: string) {
    setDraft((current) => ({ ...current, [field]: value }));
  }

  return (
    <form
      className="filters"
      onSubmit={(event) => {
        event.preventDefault();
        onApply(draft);
      }}
    >
      <label>
        <span>Platform</span>
        <select value={draft.platform} onChange={(event) => update("platform", event.target.value)}>
          <option value="">all</option>
          <option value="facebook">facebook</option>
          <option value="google_ads">google_ads</option>
          <option value="tiktok_ads">tiktok_ads</option>
        </select>
      </label>
      <label>
        <span>Account</span>
        <input value={draft.accountId} onChange={(event) => update("accountId", event.target.value)} placeholder="248-390-1805" />
      </label>
      <label>
        <span>Date From</span>
        <input type="date" value={draft.dateFrom} onChange={(event) => update("dateFrom", event.target.value)} />
      </label>
      <label>
        <span>Date To</span>
        <input type="date" value={draft.dateTo} onChange={(event) => update("dateTo", event.target.value)} />
      </label>
      <label>
        <span>Entity</span>
        <select value={draft.entityLevel} onChange={(event) => update("entityLevel", event.target.value)}>
          <option value="">all</option>
          <option value="campaign">campaign</option>
          <option value="ad_group">ad_group</option>
          <option value="ad">ad</option>
        </select>
      </label>
      <label>
        <span>Country</span>
        <input value={draft.country} onChange={(event) => update("country", event.target.value)} placeholder="US" />
      </label>
      <label>
        <span>OS</span>
        <input value={draft.os} onChange={(event) => update("os", event.target.value)} placeholder="ios" />
      </label>
      <label>
        <span>Device</span>
        <input value={draft.device} onChange={(event) => update("device", event.target.value)} placeholder="MOBILE" />
      </label>
      <label>
        <span>Network</span>
        <input value={draft.network} onChange={(event) => update("network", event.target.value)} placeholder="SEARCH" />
      </label>
      <label>
        <span>Campaign ID</span>
        <input value={draft.campaignId} onChange={(event) => update("campaignId", event.target.value)} />
      </label>
      <label>
        <span>Ad Group ID</span>
        <input value={draft.adGroupId} onChange={(event) => update("adGroupId", event.target.value)} />
      </label>
      <label>
        <span>Ad ID</span>
        <input value={draft.adId} onChange={(event) => update("adId", event.target.value)} />
      </label>
      <label>
        <span>Detail Limit</span>
        <input value={draft.detailLimit} onChange={(event) => update("detailLimit", event.target.value)} inputMode="numeric" />
      </label>
      <label>
        <span>Match Type</span>
        <input value={draft.matchType} onChange={(event) => update("matchType", event.target.value)} placeholder="EXACT" />
      </label>
      <label>
        <span>Search Term</span>
        <input value={draft.searchTerm} onChange={(event) => update("searchTerm", event.target.value)} placeholder="brand" />
      </label>
      <div className="filter-actions">
        <button type="submit">
          <Search size={16} />
          Apply
        </button>
        <button type="button" className="secondary" onClick={() => onApply(defaultFilters)}>
          <RotateCcw size={16} />
          Reset
        </button>
      </div>
    </form>
  );
}
