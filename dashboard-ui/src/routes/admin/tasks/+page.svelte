<script lang="ts">
  import { tasksAPI } from "$lib/api";
  import type {
    Ecosystem,
    CollectionTaskStatus,
    CollectionTask,
  } from "$lib/types/api";
  import { AlertCircle, Loader2, Download } from "lucide-svelte";

  const ecosystems: Ecosystem[] = ["npm", "go", "cargo", "pypi"];

  const statusColors: Record<CollectionTaskStatus, string> = {
    pending: "status-pending",
    running: "status-running",
    succeeded: "status-succeeded",
    failed: "status-failed",
    cancelled: "status-cancelled",
  };

  let selectedEcosystem = $state<Ecosystem | "">("");
  let tasks = $state<CollectionTask[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let currentPage = $state(1);
  let pageSize = $state(50);
  let downloadingTask = $state<number | null>(null);

  async function loadTasks() {
    loading = true;
    error = null;
    try {
      const response = await tasksAPI.list({
        ecosystem: selectedEcosystem || undefined,
        page: currentPage,
        page_size: pageSize,
      });
      tasks = response.items;
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load tasks";
    } finally {
      loading = false;
    }
  }

  async function downloadArtifact(taskId: number) {
    downloadingTask = taskId;
    try {
      const response = await tasksAPI.downloadArtifact(taskId);
      if (!response.ok) {
        throw new Error("Failed to download artifact");
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `task-${taskId}-behavior.jsonl`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch {
      alert("Failed to download artifact");
    } finally {
      downloadingTask = null;
    }
  }

  $effect(() => {
    loadTasks();
  });
</script>

<div class="page">
  <div class="page-header">
    <h1 class="page-title">Collection Tasks</h1>
    <p class="page-subtitle">View and manage collection tasks</p>
  </div>

  <div class="filter-bar">
    <select
      bind:value={selectedEcosystem}
      onchange={loadTasks}
      class="ecosystem-select"
    >
      <option value="">All ecosystems</option>
      {#each ecosystems as eco}
        <option value={eco}>{eco.toUpperCase()}</option>
      {/each}
    </select>
  </div>

  {#if loading}
    <div class="loading-container">
      <Loader2 class="spinner" />
    </div>
  {:else if error}
    <div class="error-banner">
      <div class="error-content">
        <AlertCircle class="icon-error" />
        <span>{error}</span>
      </div>
    </div>
  {:else if tasks.length === 0}
    <div class="empty-state">
      <p class="empty-text">No tasks found</p>
      <p class="empty-hint">Trigger a scan from the packages page</p>
    </div>
  {:else}
    <div class="table-wrapper">
      <table class="tasks-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Package</th>
            <th>Version</th>
            <th>Status</th>
            <th>Created</th>
            <th class="col-actions">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each tasks as task}
            <tr>
              <td class="cell-mono">{task.id}</td>
              <td>
                <span class="eco-pill">
                  {task.ecosystem}
                </span>
                <span class="task-identifier">{task.identifier}</span>
              </td>
              <td class="cell-mono">{task.version}</td>
              <td>
                <span class="status-badge {statusColors[task.status]}">
                  {task.status}
                </span>
              </td>
              <td class="cell-date">
                {new Date(task.created_at).toLocaleString()}
              </td>
              <td class="col-actions">
                {#if task.has_artifact}
                  <button
                    onclick={() => downloadArtifact(task.id)}
                    disabled={downloadingTask === task.id}
                    class="artifact-button"
                  >
                    {#if downloadingTask === task.id}
                      <Loader2 class="icon-xs-spin" />
                    {:else}
                      <Download class="icon-xs" />
                    {/if}
                    Artifact
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .page {
    padding: 2rem;
  }

  .page-header {
    margin-bottom: 1.5rem;
  }

  .page-title {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .page-subtitle {
    color: var(--text-secondary);
  }

  .filter-bar {
    display: flex;
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .ecosystem-select {
    height: 2.5rem;
    padding: 0 0.75rem;
    font-size: 0.875rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    cursor: pointer;
    appearance: auto;
  }

  .loading-container {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 3rem 0;
  }

  .loading-container :global(.spinner) {
    width: 2rem;
    height: 2rem;
    color: var(--text-secondary);
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .error-banner {
    border-radius: 8px;
    border: 1px solid rgba(220, 38, 38, 0.2);
    background: rgba(220, 38, 38, 0.08);
    padding: 1rem;
  }

  .error-content {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: #dc2626;
  }

  .error-content :global(.icon-error) {
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
  }

  .empty-state {
    border-radius: 8px;
    border: 2px dashed var(--border);
    padding: 3rem;
    text-align: center;
  }

  .empty-text {
    color: var(--text-secondary);
  }

  .empty-hint {
    margin-top: 0.25rem;
    font-size: 0.875rem;
    color: var(--text-secondary);
    opacity: 0.7;
  }

  .table-wrapper {
    border-radius: 8px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    overflow: auto;
  }

  .tasks-table {
    width: 100%;
    border-collapse: collapse;
  }

  .tasks-table thead tr {
    border-bottom: 1px solid var(--border);
    background: var(--bg-secondary);
  }

  .tasks-table th {
    padding: 0.75rem 1rem;
    text-align: left;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .tasks-table tbody tr {
    border-bottom: 1px solid var(--border);
    transition: background 0.1s;
  }

  .tasks-table tbody tr:last-child {
    border-bottom: none;
  }

  .tasks-table tbody tr:hover {
    background: var(--bg-secondary);
  }

  .tasks-table td {
    padding: 0.75rem 1rem;
    color: var(--text-primary);
  }

  .cell-mono {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.875rem;
  }

  .cell-date {
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .col-actions {
    text-align: right;
  }

  .eco-pill {
    display: inline-block;
    padding: 0.125rem 0.5rem;
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    border-radius: 999px;
    background: var(--bg-secondary);
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .task-identifier {
    margin-left: 0.5rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .status-badge {
    display: inline-block;
    padding: 0.125rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 500;
    border-radius: 999px;
  }

  .status-pending {
    background: rgba(234, 179, 8, 0.15);
    color: #a16207;
  }

  .status-running {
    background: rgba(37, 99, 235, 0.15);
    color: #1d4ed8;
  }

  .status-succeeded {
    background: rgba(22, 163, 74, 0.15);
    color: #15803d;
  }

  .status-failed {
    background: rgba(220, 38, 38, 0.15);
    color: #dc2626;
  }

  .status-cancelled {
    background: rgba(107, 114, 128, 0.15);
    color: #4b5563;
  }

  .artifact-button {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.25rem 0.75rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 6px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    cursor: pointer;
    transition: background 0.15s;
  }

  .artifact-button:hover {
    background: var(--bg-primary);
  }

  .artifact-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .artifact-button :global(.icon-xs) {
    width: 0.75rem;
    height: 0.75rem;
  }

  .artifact-button :global(.icon-xs-spin) {
    width: 0.75rem;
    height: 0.75rem;
    animation: spin 1s linear infinite;
  }
</style>
