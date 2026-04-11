<script lang="ts">
  import { packagesAPI } from "$lib/api";
  import type { Ecosystem, ReviewQueueItem } from "$lib/types/api";
  import {
    AlertCircle,
    CheckCircle2,
    Clock,
    Loader2,
    ShieldAlert,
    XCircle,
  } from "lucide-svelte";

  const ecosystems: Ecosystem[] = ["npm", "go", "cargo", "pypi"];

  const ecosystemColors: Record<Ecosystem, string> = {
    cargo: "eco-cargo",
    go: "eco-go",
    npm: "eco-npm",
    pypi: "eco-pypi",
  };

  type StatusFilter = "all" | "pending" | "approved" | "rejected";

  let selectedEcosystem = $state<string>("");
  let selectedStatus = $state<StatusFilter>("all");
  let items = $state<ReviewQueueItem[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function loadQueue() {
    loading = true;
    error = null;
    try {
      const params: { ecosystem?: string; status?: string } = {};
      if (selectedEcosystem) params.ecosystem = selectedEcosystem;
      if (selectedStatus !== "all") params.status = selectedStatus;
      const response = await packagesAPI.reviewQueue(params);
      items = response.items;
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load review queue";
    } finally {
      loading = false;
    }
  }

  function reviewStatus(
    item: ReviewQueueItem,
  ): "approved" | "pending" | "rejected" {
    if (item.manually_approved === null) return "pending";
    return item.manually_approved ? "approved" : "rejected";
  }

  $effect(() => {
    loadQueue();
  });
</script>

<div class="page">
  <div class="page-header">
    <div>
      <h1 class="page-title">Review Queue</h1>
      <p class="page-subtitle">
        Package versions that failed behavioral analysis
      </p>
    </div>
  </div>

  <div class="filter-bar">
    <select
      bind:value={selectedEcosystem}
      onchange={loadQueue}
      class="filter-select"
    >
      <option value="">All Ecosystems</option>
      {#each ecosystems as eco}
        <option value={eco}>{eco.toUpperCase()}</option>
      {/each}
    </select>

    <select
      bind:value={selectedStatus}
      onchange={loadQueue}
      class="filter-select"
    >
      <option value="all">All Statuses</option>
      <option value="pending">Pending</option>
      <option value="approved">Approved</option>
      <option value="rejected">Rejected</option>
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
  {:else if items.length === 0}
    <div class="empty-state">
      <ShieldAlert class="empty-icon" />
      <p class="empty-text">No items in the review queue</p>
      <p class="empty-hint">
        Packages that fail behavioral analysis will appear here
      </p>
    </div>
  {:else}
    <div class="table-wrapper">
      <table class="queue-table">
        <thead>
          <tr>
            <th>Package</th>
            <th>Version</th>
            <th>Status</th>
            <th>Comment</th>
          </tr>
        </thead>
        <tbody>
          {#each items as item}
            {@const status = reviewStatus(item)}
            {@const eco = (item.ecosystem as Ecosystem) ?? "npm"}
            {@const colorClass = ecosystemColors[eco] ?? ecosystemColors.npm}
            <tr>
              <td>
                <a
                  href="/admin/packages/{item.ecosystem}/{encodeURIComponent(
                    item.identifier,
                  )}/behavior?version={encodeURIComponent(item.version)}"
                  class="package-link"
                >
                  <span class="eco-badge {colorClass}">
                    {item.ecosystem}
                  </span>
                  <span class="package-name">{item.identifier}</span>
                </a>
              </td>
              <td class="cell-version">
                <span class="version-text">{item.version}</span>
                {#if item.is_latest}
                  <span class="badge-latest">latest</span>
                {/if}
              </td>
              <td>
                {#if status === "approved"}
                  <span class="status-badge status-approved">
                    <CheckCircle2 class="status-icon" />
                    Approved
                  </span>
                {:else if status === "rejected"}
                  <span class="status-badge status-rejected">
                    <XCircle class="status-icon" />
                    Rejected
                  </span>
                {:else}
                  <span class="status-badge status-pending">
                    <Clock class="status-icon" />
                    Pending
                  </span>
                {/if}
              </td>
              <td class="cell-comment">
                {#if item.review_comment}
                  <span class="comment-text">{item.review_comment}</span>
                {:else}
                  <span class="comment-empty">--</span>
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
    gap: 0.75rem;
    margin-bottom: 1.5rem;
  }

  .filter-select {
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
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
  }

  .empty-state :global(.empty-icon) {
    width: 2.5rem;
    height: 2.5rem;
    color: var(--text-secondary);
    opacity: 0.5;
  }

  .empty-text {
    color: var(--text-secondary);
    font-weight: 500;
  }

  .empty-hint {
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

  .queue-table {
    width: 100%;
    border-collapse: collapse;
  }

  .queue-table thead tr {
    border-bottom: 1px solid var(--border);
    background: var(--bg-secondary);
  }

  .queue-table th {
    padding: 0.75rem 1rem;
    text-align: left;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .queue-table tbody tr {
    border-bottom: 1px solid var(--border);
    transition: background 0.1s;
  }

  .queue-table tbody tr:last-child {
    border-bottom: none;
  }

  .queue-table tbody tr:hover {
    background: var(--bg-secondary);
  }

  .queue-table td {
    padding: 0.75rem 1rem;
  }

  .package-link {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    text-decoration: none;
    transition: color 0.15s;
  }

  .package-link:hover .package-name {
    color: var(--accent);
  }

  .eco-badge {
    display: inline-block;
    flex-shrink: 0;
    padding: 0.125rem 0.5rem;
    font-size: 0.65rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.025em;
    border-radius: 999px;
    border: 1px solid;
  }

  .eco-npm {
    border-color: rgba(252, 165, 165, 0.4);
    color: #b91c1c;
    background: #fef2f2;
  }

  .eco-go {
    border-color: rgba(103, 232, 249, 0.4);
    color: #0e7490;
    background: #ecfeff;
  }

  .eco-cargo {
    border-color: rgba(253, 186, 116, 0.4);
    color: #c2410c;
    background: #fff7ed;
  }

  .eco-pypi {
    border-color: rgba(147, 197, 253, 0.4);
    color: #1d4ed8;
    background: #eff6ff;
  }

  .package-name {
    font-weight: 600;
    color: var(--text-primary);
    transition: color 0.15s;
  }

  .cell-version {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  }

  .version-text {
    color: var(--text-primary);
  }

  .badge-latest {
    display: inline-block;
    margin-left: 0.5rem;
    padding: 0.125rem 0.5rem;
    font-size: 0.7rem;
    font-weight: 600;
    font-family:
      system-ui,
      -apple-system,
      sans-serif;
    border-radius: 999px;
    background: rgba(22, 163, 74, 0.15);
    color: #15803d;
  }

  .status-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.25rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    border-radius: 999px;
  }

  .status-badge :global(.status-icon) {
    width: 0.875rem;
    height: 0.875rem;
  }

  .status-approved {
    background: rgba(22, 163, 74, 0.12);
    color: #15803d;
  }

  .status-rejected {
    background: rgba(220, 38, 38, 0.12);
    color: #dc2626;
  }

  .status-pending {
    background: rgba(234, 179, 8, 0.12);
    color: #a16207;
  }

  .cell-comment {
    max-width: 20rem;
  }

  .comment-text {
    font-size: 0.875rem;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    display: block;
  }

  .comment-empty {
    color: var(--text-secondary);
    opacity: 0.4;
  }
</style>
