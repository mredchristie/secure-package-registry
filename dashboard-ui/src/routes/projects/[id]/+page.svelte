<script lang="ts">
  import { page } from "$app/stores";
  import { projectsAPI } from "$lib/api";
  import type {
    DependencyType,
    Project,
    ProjectDependency,
    ProjectSummaryRow,
  } from "$lib/types/api";
  import {
    AlertCircle,
    ArrowLeft,
    Loader2,
    Package,
    ShieldAlert,
    ShieldCheck,
  } from "lucide-svelte";

  const projectId = $derived(Number($page.params.id));

  let project = $state<Project | null>(null);
  let deps = $state<ProjectDependency[]>([]);
  let summary = $state<ProjectSummaryRow[]>([]);

  let loadingProject = $state(false);
  let loadingDeps = $state(false);
  let loadingSummary = $state(false);
  let error = $state<string | null>(null);

  let typeFilter = $state<DependencyType | "">("");
  let sortByChecks = $state<"asc" | "desc" | null>(null);

  async function loadProject() {
    loadingProject = true;
    error = null;
    try {
      project = await projectsAPI.get(projectId);
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load project";
    } finally {
      loadingProject = false;
    }
  }

  async function loadDeps() {
    loadingDeps = true;
    try {
      const response = await projectsAPI.dependencies(
        projectId,
        typeFilter || undefined,
      );
      deps = response.items;
    } catch (e) {
      if (!error) {
        error = e instanceof Error ? e.message : "Failed to load dependencies";
      }
    } finally {
      loadingDeps = false;
    }
  }

  async function loadSummary() {
    loadingSummary = true;
    try {
      const response = await projectsAPI.summary(projectId);
      summary = response.summary;
    } catch {
      // Non-fatal: summary is supplementary.
    } finally {
      loadingSummary = false;
    }
  }

  function formatDate(dateStr?: string): string {
    if (!dateStr) return "-";
    try {
      return new Date(dateStr).toLocaleDateString("en-GB", {
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        month: "short",
        year: "numeric",
      });
    } catch {
      return dateStr.slice(0, 19);
    }
  }

  // Count of passed checks (only counts true, excludes null/pending).
  function checksPassedCount(dep: ProjectDependency): number {
    return (
      (dep.has_attestation === true ? 1 : 0) +
      (dep.has_oss_rebuild === true ? 1 : 0) +
      (dep.behavior_passed === true ? 1 : 0)
    );
  }

  // Count of pending checks (null values).
  function checksPendingCount(dep: ProjectDependency): number {
    return (
      (dep.has_attestation === null ? 1 : 0) +
      (dep.has_oss_rebuild === null ? 1 : 0) +
      (dep.behavior_passed === null ? 1 : 0)
    );
  }

  function toggleSort() {
    if (sortByChecks === null) sortByChecks = "asc";
    else if (sortByChecks === "asc") sortByChecks = "desc";
    else sortByChecks = null;
  }

  // Aggregate totals across direct + transitive rows.
  const totalDeps = $derived(summary.reduce((n, s) => n + s.total, 0));
  const directRow = $derived(
    summary.find((s) => s.dependency_type === "direct"),
  );
  const transitiveRow = $derived(
    summary.find((s) => s.dependency_type === "transitive"),
  );

  // Count deps that fail ALL 3 checks (0 of 3 passed, no pending).
  const totalFailedAll = $derived(
    deps.filter(
      (d) => checksPendingCount(d) === 0 && checksPassedCount(d) === 0,
    ).length,
  );

  // Sorted deps list.
  const sortedDeps = $derived.by(() => {
    if (sortByChecks === null) return deps;
    const sorted = [...deps];
    sorted.sort((a, b) => {
      const diff = checksPassedCount(a) - checksPassedCount(b);
      return sortByChecks === "asc" ? diff : -diff;
    });
    return sorted;
  });

  // Initial data load
  $effect(() => {
    // Re-run when projectId changes (handles SvelteKit navigations)
    if (projectId) {
      loadProject();
      loadDeps();
      loadSummary();
    }
  });

  // Reload deps when filter changes (but not on initial mount)
  let filterInitialized = false;
  $effect(() => {
    // Subscribe to typeFilter
    typeFilter;
    if (filterInitialized) {
      loadDeps();
    }
    filterInitialized = true;
  });
</script>

<svelte:head>
  <title>{project?.name ?? "Project"} - SPR</title>
</svelte:head>

<div class="page">
  <!-- Header -->
  <div class="page-header">
    <a href="/projects" class="back-link">
      <ArrowLeft class="icon-sm" />
      Back to Projects
    </a>
  </div>

  {#if loadingProject && !project}
    <div class="loading-container">
      <Loader2 class="spinner" />
    </div>
  {:else if error && !project}
    <div class="error-banner">
      <div class="error-content">
        <AlertCircle class="icon-error" />
        <span>{error}</span>
      </div>
    </div>
  {:else if project}
    <!-- Project info -->
    <div class="project-info">
      <div class="info-left">
        <h1 class="project-title">{project.name}</h1>
        <div class="info-meta">
          <span class="source-badge">{project.source_type}</span>
          <span class="meta-text">
            Updated {formatDate(project.updated_at)}
          </span>
        </div>
      </div>
    </div>

    <!-- Summary cards -->
    {#if summary.length > 0}
      <div class="summary-grid">
        <div class="summary-card">
          <div class="summary-icon-wrap summary-total">
            <Package class="summary-icon" />
          </div>
          <div class="summary-data">
            <span class="summary-value">{totalDeps}</span>
            <span class="summary-label">Total Dependencies</span>
            <span class="summary-detail">
              {directRow?.total ?? 0} direct, {transitiveRow?.total ?? 0} transitive
            </span>
          </div>
        </div>

        <div
          class="summary-card"
          class:summary-card-danger={totalFailedAll > 0}
        >
          <div class="summary-icon-wrap summary-failed">
            <ShieldAlert class="summary-icon" />
          </div>
          <div class="summary-data">
            <span class="summary-value">
              {totalFailedAll}<span class="summary-of">/{totalDeps}</span>
            </span>
            <span class="summary-label">Failed All Checks</span>
            <span class="summary-detail">
              No attestation, rebuild, or clean behavior
            </span>
          </div>
        </div>
      </div>
    {:else if loadingSummary}
      <div class="loading-container-sm">
        <Loader2 class="spinner-sm" />
      </div>
    {/if}

    <!-- Dependencies table -->
    <div class="deps-section">
      <div class="deps-header">
        <h2 class="deps-title">Dependencies</h2>
        <select bind:value={typeFilter} class="type-select">
          <option value="">All types</option>
          <option value="direct">Direct</option>
          <option value="transitive">Transitive</option>
        </select>
      </div>

      {#if loadingDeps}
        <div class="loading-container">
          <Loader2 class="spinner" />
        </div>
      {:else if deps.length === 0}
        <div class="empty-state">
          <p class="empty-text">No dependencies found</p>
        </div>
      {:else}
        <div class="table-wrapper">
          <table class="deps-table">
            <thead>
              <tr>
                <th>Package</th>
                <th>Version</th>
                <th>Type</th>
                <th>
                  <button class="sort-btn" onclick={toggleSort}>
                    Checks
                    {#if sortByChecks === "asc"}
                      <span class="sort-arrow">&#9650;</span>
                    {:else if sortByChecks === "desc"}
                      <span class="sort-arrow">&#9660;</span>
                    {:else}
                      <span class="sort-arrow sort-arrow-idle">&#9650;</span>
                    {/if}
                  </button>
                </th>
              </tr>
            </thead>
            <tbody>
              {#each sortedDeps as dep}
                {@const passed = checksPassedCount(dep)}
                {@const pending = checksPendingCount(dep)}
                <tr>
                  <td>
                    <span class="eco-pill">{dep.ecosystem}</span>
                    <span class="dep-name">{dep.identifier}</span>
                  </td>
                  <td class="cell-mono">{dep.version}</td>
                  <td>
                    <span
                      class="type-badge"
                      class:type-direct={dep.dependency_type === "direct"}
                      class:type-transitive={dep.dependency_type ===
                        "transitive"}
                    >
                      {dep.dependency_type}
                    </span>
                  </td>
                  <td>
                    {#if pending > 0}
                      <span class="checks-badge checks-pending">
                        <Loader2 class="checks-icon checks-icon-spin" />
                        Pending
                      </span>
                    {:else}
                      <span
                        class="checks-badge"
                        class:checks-none={passed === 0}
                        class:checks-partial={passed > 0 && passed < 3}
                        class:checks-all={passed === 3}
                      >
                        <ShieldCheck class="checks-icon" />
                        {passed}/3
                      </span>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .page {
    padding: 2rem;
    max-width: 60rem;
    margin: 0 auto;
  }

  .page-header {
    margin-bottom: 1.5rem;
  }

  .back-link {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    text-decoration: none;
    transition: color 0.15s;
  }

  .back-link:hover {
    color: var(--text-primary);
  }

  .back-link :global(.icon-sm) {
    width: 1rem;
    height: 1rem;
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

  .loading-container-sm {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1.5rem 0;
  }

  .loading-container-sm :global(.spinner-sm) {
    width: 1.25rem;
    height: 1.25rem;
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

  /* Project info */
  .project-info {
    margin-bottom: 1.5rem;
  }

  .project-title {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .info-meta {
    margin-top: 0.5rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .source-badge {
    display: inline-block;
    padding: 0.125rem 0.625rem;
    font-size: 0.7rem;
    font-weight: 600;
    border-radius: 999px;
    background: var(--bg-secondary);
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .meta-text {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  /* Summary cards */
  .summary-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
  }

  .summary-card {
    display: flex;
    align-items: flex-start;
    gap: 1rem;
    padding: 1rem 1.25rem;
    border-radius: 12px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  .summary-icon-wrap {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    border-radius: 8px;
    flex-shrink: 0;
  }

  .summary-icon-wrap :global(.summary-icon) {
    width: 1.25rem;
    height: 1.25rem;
  }

  .summary-total {
    background: rgba(99, 102, 241, 0.1);
    color: #6366f1;
  }

  .summary-failed {
    background: rgba(220, 38, 38, 0.1);
    color: #dc2626;
  }

  .summary-card-danger {
    border-color: rgba(220, 38, 38, 0.3);
    background: rgba(220, 38, 38, 0.04);
  }

  .summary-data {
    display: flex;
    flex-direction: column;
  }

  .summary-value {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.2;
  }

  .summary-of {
    font-size: 1rem;
    font-weight: 400;
    color: var(--text-secondary);
  }

  .summary-label {
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-secondary);
    margin-top: 0.125rem;
  }

  .summary-detail {
    font-size: 0.7rem;
    color: var(--text-secondary);
    opacity: 0.7;
    margin-top: 0.125rem;
  }

  /* Dependencies section */
  .deps-section {
    margin-top: 0.5rem;
  }

  .deps-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
  }

  .deps-title {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .type-select {
    height: 2.25rem;
    padding: 0 0.75rem;
    font-size: 0.8rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    cursor: pointer;
    appearance: auto;
  }

  .empty-state {
    border-radius: 8px;
    border: 2px dashed var(--border);
    padding: 2rem;
    text-align: center;
  }

  .empty-text {
    color: var(--text-secondary);
  }

  .table-wrapper {
    border-radius: 8px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    overflow: auto;
  }

  .deps-table {
    width: 100%;
    border-collapse: collapse;
  }

  .deps-table thead tr {
    border-bottom: 1px solid var(--border);
    background: var(--bg-secondary);
  }

  .deps-table th {
    padding: 0.75rem 1rem;
    text-align: left;
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .deps-table tbody tr {
    border-bottom: 1px solid var(--border);
    transition: background 0.1s;
  }

  .deps-table tbody tr:last-child {
    border-bottom: none;
  }

  .deps-table tbody tr:hover {
    background: var(--bg-secondary);
  }

  .deps-table td {
    padding: 0.625rem 1rem;
    color: var(--text-primary);
    font-size: 0.875rem;
  }

  .cell-mono {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.8rem;
  }

  .eco-pill {
    display: inline-block;
    padding: 0.0625rem 0.4375rem;
    font-size: 0.65rem;
    font-weight: 600;
    text-transform: uppercase;
    border-radius: 999px;
    background: var(--bg-secondary);
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .dep-name {
    margin-left: 0.5rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .type-badge {
    display: inline-block;
    padding: 0.125rem 0.5rem;
    font-size: 0.7rem;
    font-weight: 500;
    border-radius: 999px;
  }

  .type-direct {
    background: rgba(99, 102, 241, 0.12);
    color: #6366f1;
  }

  .type-transitive {
    background: rgba(107, 114, 128, 0.12);
    color: #6b7280;
  }

  /* Sort button in table header */
  .sort-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    background: none;
    border: none;
    padding: 0;
    font: inherit;
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-secondary);
    cursor: pointer;
    transition: color 0.15s;
  }

  .sort-btn:hover {
    color: var(--text-primary);
  }

  .sort-arrow {
    font-size: 0.65rem;
    line-height: 1;
  }

  .sort-arrow-idle {
    opacity: 0.3;
  }

  /* Checks badge in dependency table */
  .checks-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.125rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 600;
    border-radius: 999px;
  }

  .checks-badge :global(.checks-icon) {
    width: 0.875rem;
    height: 0.875rem;
  }

  .checks-none {
    background: rgba(220, 38, 38, 0.1);
    color: #dc2626;
  }

  .checks-partial {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
  }

  .checks-all {
    background: rgba(16, 185, 129, 0.1);
    color: #10b981;
  }

  .checks-pending {
    background: rgba(107, 114, 128, 0.1);
    color: #6b7280;
  }

  .checks-pending :global(.checks-icon) {
    animation: spin 1s linear infinite;
  }
</style>
