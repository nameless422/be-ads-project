import { Play, RefreshCw, RotateCw, Square, TestTube2 } from "lucide-react";
import { useEffect, useState } from "react";
import { loadDeadLetters, runLocalStackAction } from "../api/client";
import type { BIDashboardData, DeadLetterView, LocalCommandResult } from "../api/types";
import { DataTable } from "../components/DataTable";
import { KpiCard } from "../components/KpiCard";
import { formatDate } from "../utils/metrics";

type ControlPageProps = {
  data: BIDashboardData;
  onRefresh: () => void;
};

const actionButtons = [
  { action: "start", label: "Start", icon: <Play size={16} /> },
  { action: "stop", label: "Stop", icon: <Square size={16} /> },
  { action: "restart", label: "Restart", icon: <RotateCw size={16} /> },
  { action: "verify", label: "Verify", icon: <TestTube2 size={16} /> },
  { action: "start-infra", label: "Infra Up", icon: <Play size={16} /> },
  { action: "stop-infra", label: "Infra Down", icon: <Square size={16} /> },
  { action: "start-workers", label: "Workers Up", icon: <Play size={16} /> },
  { action: "stop-workers", label: "Workers Down", icon: <Square size={16} /> },
  { action: "restart-collector", label: "Restart Collector", icon: <RotateCw size={16} /> },
];

export function ControlPage({ data, onRefresh }: ControlPageProps) {
  const overview = data.controlOverview;
  const [deadLetters, setDeadLetters] = useState<DeadLetterView[]>([]);
  const [actionResult, setActionResult] = useState<LocalCommandResult | null>(null);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadDeadLetters().then(setDeadLetters).catch(() => setDeadLetters([]));
  }, []);

  async function runAction(action: string, role?: string) {
    setBusyAction(role ? `${action}-${role}` : action);
    setError(null);
    try {
      const result = await runLocalStackAction(action, role);
      setActionResult(result);
      onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyAction(null);
    }
  }

  return (
    <>
      <section className="kpi-grid compact">
        <KpiCard label="Worker Leases" value={overview?.WorkerLeases?.length ?? 0} />
        <KpiCard label="Shard Assignments" value={overview?.ShardAssignments?.length ?? 0} />
        <KpiCard label="Raw Records" value={overview?.RawRecordCount ?? 0} />
        <KpiCard label="Outbox Pending" value={overview?.OutboxPending ?? 0} tone={(overview?.OutboxPending ?? 0) > 0 ? "warn" : "default"} />
        <KpiCard label="Outbox Published" value={overview?.OutboxPublished ?? 0} tone="good" />
        <KpiCard label="DLQ" value={overview?.DeadLetterCount ?? 0} tone={(overview?.DeadLetterCount ?? 0) > 0 ? "warn" : "default"} />
      </section>

      <section className="panel control-actions">
        <div className="panel-head">
          <h2>Local Stack Actions</h2>
          <span>{overview?.LocalStack?.enabled ? "local controls enabled" : "local controls disabled"}</span>
        </div>
        <div className="action-grid">
          {actionButtons.map((item) => (
            <button key={item.action} type="button" onClick={() => runAction(item.action)} disabled={busyAction !== null}>
              {item.icon}
              {busyAction === item.action ? "Running" : item.label}
            </button>
          ))}
          <button type="button" onClick={() => runAction("add-worker", "collector")} disabled={busyAction !== null}>
            <Play size={16} />
            Add Collector
          </button>
          <button type="button" onClick={() => runAction("remove-worker", "collector")} disabled={busyAction !== null}>
            <Square size={16} />
            Remove Collector
          </button>
          <button type="button" onClick={() => runAction("add-worker", "transformer")} disabled={busyAction !== null}>
            <Play size={16} />
            Add Transformer
          </button>
          <button type="button" onClick={() => runAction("remove-worker", "transformer")} disabled={busyAction !== null}>
            <Square size={16} />
            Remove Transformer
          </button>
          <button type="button" className="secondary" onClick={onRefresh} disabled={busyAction !== null}>
            <RefreshCw size={16} />
            Refresh Data
          </button>
        </div>
        {error ? <pre className="command-output warn-output">{error}</pre> : null}
        {actionResult ? <pre className="command-output">{actionResult.output || actionResult.error || `${actionResult.action}: ${actionResult.success}`}</pre> : null}
      </section>

      <section className="two-column">
        <DataTable
          title="Worker Leases"
          rows={overview?.WorkerLeases ?? []}
          columns={[
            { key: "role", header: "Role", render: (row) => <span className="badge">{row.WorkerRole}</span> },
            { key: "id", header: "Worker ID", render: (row) => row.WorkerID },
            { key: "platform", header: "Platform", render: (row) => row.PlatformScope || "-" },
            { key: "capacity", header: "Capacity", render: (row) => row.Capacity },
            { key: "last", header: "Last Seen", render: (row) => row.LastSeenAt?.slice(0, 19).replace("T", " ") },
            { key: "expires", header: "Expires", render: (row) => row.ExpiresAt?.slice(0, 19).replace("T", " ") },
          ]}
          getRowKey={(row, index) => `${row.WorkerRole}-${row.WorkerID}-${index}`}
          emptyText="No worker lease data"
        />
        <DataTable
          title="Shard Assignments"
          rows={overview?.ShardAssignments ?? []}
          columns={[
            { key: "role", header: "Role", render: (row) => row.WorkerRole },
            { key: "platform", header: "Platform", render: (row) => <span className="badge">{row.Platform}</span> },
            { key: "shard", header: "Shard", render: (row) => row.ShardID },
            { key: "worker", header: "Worker", render: (row) => row.WorkerID },
            { key: "updated", header: "Updated", render: (row) => row.UpdatedAt?.slice(0, 19).replace("T", " ") },
          ]}
          getRowKey={(row, index) => `${row.WorkerRole}-${row.Platform}-${row.ShardID}-${index}`}
          emptyText="No shard assignment data"
        />
      </section>

      <section className="two-column">
        <DataTable
          title="Local Ports"
          rows={overview?.LocalStack?.ports ?? []}
          columns={[
            { key: "name", header: "Name", render: (row) => row.name },
            { key: "port", header: "Port", render: (row) => row.port },
            { key: "state", header: "State", render: (row) => <span className={`badge ${row.state !== "listening" ? "warn" : ""}`}>{row.state}</span> },
            { key: "detail", header: "Detail", render: (row) => row.detail || "-" },
          ]}
          getRowKey={(row) => `${row.name}-${row.port}`}
          emptyText="No local port status"
        />
        <DataTable
          title="Dead Letters"
          rows={deadLetters}
          columns={[
            { key: "seq", header: "Seq", render: (row) => row.StreamSequence },
            { key: "kind", header: "Kind", render: (row) => row.Kind },
            { key: "platform", header: "Platform", render: (row) => row.Platform || "-" },
            { key: "subject", header: "Subject", render: (row) => row.Subject },
            { key: "delivery", header: "Delivery", render: (row) => row.DeliveryCount },
            { key: "failed", header: "Failed At", render: (row) => formatDate(row.FailedAt) },
            { key: "error", header: "Error", render: (row) => row.ErrorMessage || "-" },
          ]}
          getRowKey={(row, index) => `${row.StreamSequence}-${index}`}
          emptyText="No DLQ data"
        />
      </section>
    </>
  );
}
