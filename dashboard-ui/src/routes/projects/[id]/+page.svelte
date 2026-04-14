<script lang="ts">
  import { page } from "$app/stores";
  import { projectsAPI } from "$lib/api";
  import type {
    CreateAPIKeyResponse,
    DependencyType,
    Project,
    ProjectAPIKey,
    ProjectDependency,
    ProjectPolicy,
    ProjectSummaryRow,
  } from "$lib/types/api";
  import {
    AlertCircle,
    ArrowLeft,
    Check,
    ClipboardCopy,
    Key,
    Loader2,
    Package,
    Plus,
    Settings,
    ShieldAlert,
    ShieldCheck,
    Trash2,
    X,
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

  // Policy state
  let policy = $state<ProjectPolicy | null>(null);
  let loadingPolicy = $state(false);
  let savingPolicy = $state(false);
  let policyError = $state<string | null>(null);
  let policySaved = $state(false);

  // API keys state
  let apiKeys = $state<ProjectAPIKey[]>([]);
  let loadingKeys = $state(false);
  let newKeyName = $state("");
  let creatingKey = $state(false);
  let createdKey = $state<CreateAPIKeyResponse | null>(null);
  let keyCopied = $state(false);
  let keyError = $state<string | null>(null);
  let deletingKeyId = $state<string | null>(null);

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

  async function loadPolicy() {
    loadingPolicy = true;
    policyError = null;
    try {
      policy = await projectsAPI.getPolicy(projectId);
    } catch (e) {
      policyError = e instanceof Error ? e.message : "Failed to load policy";
    } finally {
      loadingPolicy = false;
    }
  }

  async function updatePolicy(field: keyof ProjectPolicy, value: boolean) {
    if (!policy) return;
    savingPolicy = true;
    policyError = null;
    policySaved = false;
    try {
      policy = await projectsAPI.updatePolicy(projectId, {
        [field]: value,
      });
      policySaved = true;
      setTimeout(() => (policySaved = false), 2000);
    } catch (e) {
      policyError = e instanceof Error ? e.message : "Failed to update policy";
    } finally {
      savingPolicy = false;
    }
  }

  async function loadAPIKeys() {
    loadingKeys = true;
    keyError = null;
    try {
      const response = await projectsAPI.listAPIKeys(projectId);
      apiKeys = response.items;
    } catch (e) {
      keyError = e instanceof Error ? e.message : "Failed to load API keys";
    } finally {
      loadingKeys = false;
    }
  }

  async function createAPIKey() {
    if (!newKeyName.trim()) return;
    creatingKey = true;
    keyError = null;
    try {
      createdKey = await projectsAPI.createAPIKey(projectId, newKeyName.trim());
      newKeyName = "";
      await loadAPIKeys();
    } catch (e) {
      keyError = e instanceof Error ? e.message : "Failed to create API key";
    } finally {
      creatingKey = false;
    }
  }

  async function deleteAPIKey(keyId: string) {
    deletingKeyId = keyId;
    keyError = null;
    try {
      await projectsAPI.deleteAPIKey(projectId, keyId);
      apiKeys = apiKeys.filter((k) => k.id !== keyId);
      if (createdKey?.id === keyId) createdKey = null;
    } catch (e) {
      keyError = e instanceof Error ? e.message : "Failed to delete API key";
    } finally {
      deletingKeyId = null;
    }
  }

  async function copyKey(text: string) {
    await navigator.clipboard.writeText(text);
    keyCopied = true;
    setTimeout(() => (keyCopied = false), 2000);
  }

  function dismissCreatedKey() {
    createdKey = null;
    keyCopied = false;
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

  // Total applicable checks for a dependency.
  // Score is 0-3: provenance (grouped), behavioral analysis (direct only), manual review.
  // Transitive deps don't get behavioral analysis, so max is 2 for them.
  function checksTotalCount(dep: ProjectDependency): number {
    return dep.dependency_type === "direct" ? 3 : 2;
  }

  // Count of passed checks using grouped provenance (any one = 1 point).
  function checksPassedCount(dep: ProjectDependency): number {
    const hasProvenance =
      dep.has_attestation === true ||
      dep.has_oss_rebuild === true ||
      dep.has_reproducible === true;
    return (
      (hasProvenance ? 1 : 0) +
      (dep.dependency_type === "direct" && dep.behavior_passed === true ? 1 : 0)
    );
  }

  // Count of pending checks (null values, scoped to applicable checks).
  function checksPendingCount(dep: ProjectDependency): number {
    // Provenance is pending if all three sub-checks are null
    const provenancePending =
      dep.has_attestation === null &&
      dep.has_oss_rebuild === null &&
      dep.has_reproducible === null;
    return (
      (provenancePending ? 1 : 0) +
      (dep.dependency_type === "direct" && dep.behavior_passed === null ? 1 : 0)
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

  // Count deps that fail ALL applicable checks (0 passed, no pending).
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
      loadPolicy();
      loadAPIKeys();
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

    <!-- Policy + API Keys grid -->
    <div class="settings-grid">
      <!-- Policy section -->
      <div class="settings-card">
        <div class="settings-card-header">
          <Settings class="settings-card-icon" />
          <h2 class="settings-card-title">Enforcement Policy</h2>
          {#if policySaved}
            <span class="saved-badge">
              <Check class="saved-icon" /> Saved
            </span>
          {/if}
          {#if savingPolicy}
            <Loader2 class="saving-spinner" />
          {/if}
        </div>
        <p class="settings-card-desc">
          Configure which checks must pass before the registry proxy allows
          package installation.
        </p>

        {#if loadingPolicy}
          <div class="loading-container-sm">
            <Loader2 class="spinner-sm" />
          </div>
        {:else if policyError}
          <div class="inline-error">{policyError}</div>
        {:else if policy}
          <div class="policy-toggles">
            <label class="toggle-row">
              <div class="toggle-info">
                <span class="toggle-label">Require provenance</span>
                <span class="toggle-desc"
                  >Block packages without upstream attestation or OSS rebuild
                  verification</span
                >
              </div>
              <button
                class="toggle-switch"
                class:toggle-on={policy.require_provenance}
                onclick={() =>
                  updatePolicy(
                    "require_provenance",
                    !policy!.require_provenance,
                  )}
                disabled={savingPolicy}
                aria-label="Toggle require provenance"
              >
                <span class="toggle-knob"></span>
              </button>
            </label>

            <label class="toggle-row">
              <div class="toggle-info">
                <span class="toggle-label">Require behavior analysis</span>
                <span class="toggle-desc"
                  >Block packages that haven't passed behavioral analysis
                  (sandbox testing)</span
                >
              </div>
              <button
                class="toggle-switch"
                class:toggle-on={policy.require_behavior}
                onclick={() =>
                  updatePolicy("require_behavior", !policy!.require_behavior)}
                disabled={savingPolicy}
                aria-label="Toggle require behavior analysis"
              >
                <span class="toggle-knob"></span>
              </button>
            </label>

            <label class="toggle-row">
              <div class="toggle-info">
                <span class="toggle-label">Allow manual review override</span>
                <span class="toggle-desc"
                  >Permit manually-approved packages to bypass failed checks</span
                >
              </div>
              <button
                class="toggle-switch"
                class:toggle-on={policy.allow_manual_review}
                onclick={() =>
                  updatePolicy(
                    "allow_manual_review",
                    !policy!.allow_manual_review,
                  )}
                disabled={savingPolicy}
                aria-label="Toggle allow manual review"
              >
                <span class="toggle-knob"></span>
              </button>
            </label>
          </div>
        {/if}
      </div>

      <!-- API Keys section -->
      <div class="settings-card">
        <div class="settings-card-header">
          <Key class="settings-card-icon" />
          <h2 class="settings-card-title">API Keys</h2>
        </div>
        <p class="settings-card-desc">
          Keys used to authenticate <code>npm install</code> through the SPR proxy.
        </p>

        {#if keyError}
          <div class="inline-error">{keyError}</div>
        {/if}

        <!-- Create key form -->
        <form
          class="create-key-form"
          onsubmit={(e) => {
            e.preventDefault();
            createAPIKey();
          }}
        >
          <input
            type="text"
            class="key-name-input"
            placeholder="Key name (e.g. ci-deploy)"
            bind:value={newKeyName}
            disabled={creatingKey}
          />
          <button
            type="submit"
            class="btn btn-primary btn-sm"
            disabled={creatingKey || !newKeyName.trim()}
          >
            {#if creatingKey}
              <Loader2 class="btn-icon btn-icon-spin" />
            {:else}
              <Plus class="btn-icon" />
            {/if}
            Create
          </button>
        </form>

        <!-- Newly created key banner -->
        {#if createdKey}
          <div class="created-key-banner">
            <div class="created-key-header">
              <span class="created-key-title">Key created — copy it now</span>
              <button
                class="dismiss-btn"
                onclick={dismissCreatedKey}
                aria-label="Dismiss"
              >
                <X class="dismiss-icon" />
              </button>
            </div>
            <p class="created-key-warning">This key will not be shown again.</p>
            <div class="key-display">
              <code class="key-value">{createdKey.key}</code>
              <button
                class="copy-btn"
                onclick={() => copyKey(createdKey!.key)}
                aria-label="Copy key"
              >
                {#if keyCopied}
                  <Check class="copy-icon copy-ok" />
                {:else}
                  <ClipboardCopy class="copy-icon" />
                {/if}
              </button>
            </div>
            <div class="npm-config-hint">
              <span class="hint-label">npm config:</span>
              <code class="hint-code"
                >npm config set //localhost:7002/npm/:_authToken={createdKey.key}</code
              >
            </div>
          </div>
        {/if}

        <!-- Key list -->
        {#if loadingKeys}
          <div class="loading-container-sm">
            <Loader2 class="spinner-sm" />
          </div>
        {:else if apiKeys.length === 0}
          <p class="empty-keys">No API keys yet.</p>
        {:else}
          <div class="keys-list">
            {#each apiKeys as key (key.id)}
              <div class="key-row">
                <div class="key-info">
                  <span class="key-name">{key.name}</span>
                  <code class="key-prefix">{key.prefix}...</code>
                </div>
                <div class="key-meta">
                  <span class="key-date">{formatDate(key.created_at)}</span>
                  <button
                    class="delete-key-btn"
                    onclick={() => deleteAPIKey(key.id)}
                    disabled={deletingKeyId === key.id}
                    aria-label="Delete key {key.name}"
                  >
                    {#if deletingKeyId === key.id}
                      <Loader2 class="delete-icon delete-icon-spin" />
                    {:else}
                      <Trash2 class="delete-icon" />
                    {/if}
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

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
                {@const total = checksTotalCount(dep)}
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
                        class:checks-partial={passed > 0 && passed < total}
                        class:checks-all={passed === total}
                      >
                        <ShieldCheck class="checks-icon" />
                        {passed}/{total}
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

  /* Settings grid (policy + API keys) */
  .settings-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1.5rem;
    margin-bottom: 2rem;
  }

  @media (max-width: 768px) {
    .settings-grid {
      grid-template-columns: 1fr;
    }
  }

  .settings-card {
    border-radius: 12px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    padding: 1.25rem;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  .settings-card-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .settings-card-header :global(.settings-card-icon) {
    width: 1.125rem;
    height: 1.125rem;
    color: var(--text-secondary);
  }

  .settings-card-title {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .settings-card-desc {
    font-size: 0.8rem;
    color: var(--text-secondary);
    margin-bottom: 1rem;
    line-height: 1.4;
  }

  .settings-card-desc code {
    font-size: 0.75rem;
    padding: 0.125rem 0.375rem;
    border-radius: 4px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
  }

  .inline-error {
    font-size: 0.8rem;
    color: #dc2626;
    margin-bottom: 0.75rem;
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    background: rgba(220, 38, 38, 0.08);
    border: 1px solid rgba(220, 38, 38, 0.2);
  }

  /* Policy toggles */
  .policy-toggles {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .toggle-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.625rem 0.75rem;
    border-radius: 8px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    cursor: pointer;
    transition: border-color 0.15s;
  }

  .toggle-row:hover {
    border-color: var(--accent);
  }

  .toggle-info {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .toggle-label {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .toggle-desc {
    font-size: 0.7rem;
    color: var(--text-secondary);
    line-height: 1.3;
    margin-top: 0.125rem;
  }

  .toggle-switch {
    position: relative;
    width: 2.5rem;
    height: 1.375rem;
    border-radius: 999px;
    border: none;
    background: var(--border);
    cursor: pointer;
    flex-shrink: 0;
    transition:
      background 0.2s,
      opacity 0.2s;
    padding: 0;
  }

  .toggle-switch:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .toggle-switch.toggle-on {
    background: #10b981;
  }

  .toggle-knob {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 1.125rem;
    height: 1.125rem;
    border-radius: 50%;
    background: white;
    transition: transform 0.2s;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  }

  .toggle-on .toggle-knob {
    transform: translateX(1.125rem);
  }

  .saved-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.7rem;
    font-weight: 500;
    color: #10b981;
    margin-left: auto;
  }

  .saved-badge :global(.saved-icon) {
    width: 0.75rem;
    height: 0.75rem;
  }

  .settings-card-header :global(.saving-spinner) {
    width: 0.875rem;
    height: 0.875rem;
    color: var(--text-secondary);
    animation: spin 1s linear infinite;
    margin-left: auto;
  }

  /* API keys */
  .create-key-form {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .key-name-input {
    flex: 1;
    height: 2.25rem;
    padding: 0 0.75rem;
    font-size: 0.8rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    outline: none;
    transition: border-color 0.15s;
  }

  .key-name-input::placeholder {
    color: var(--text-secondary);
    opacity: 0.6;
  }

  .key-name-input:focus {
    border-color: var(--accent);
  }

  .btn {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    font-weight: 500;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    transition:
      background 0.15s,
      opacity 0.15s;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-sm {
    height: 2.25rem;
    padding: 0 0.875rem;
    font-size: 0.8rem;
  }

  .btn-primary {
    background: var(--accent);
    color: white;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .btn :global(.btn-icon) {
    width: 0.875rem;
    height: 0.875rem;
  }

  .btn :global(.btn-icon-spin) {
    animation: spin 1s linear infinite;
  }

  /* Created key banner */
  .created-key-banner {
    border-radius: 8px;
    border: 1px solid rgba(16, 185, 129, 0.3);
    background: rgba(16, 185, 129, 0.06);
    padding: 0.875rem;
    margin-bottom: 1rem;
  }

  .created-key-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .created-key-title {
    font-size: 0.8rem;
    font-weight: 600;
    color: #10b981;
  }

  .dismiss-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 4px;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 0.15s;
  }

  .dismiss-btn:hover {
    background: var(--bg-secondary);
  }

  .dismiss-btn :global(.dismiss-icon) {
    width: 0.875rem;
    height: 0.875rem;
  }

  .created-key-warning {
    font-size: 0.7rem;
    color: var(--text-secondary);
    margin: 0.375rem 0;
  }

  .key-display {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-top: 0.5rem;
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
  }

  .key-value {
    flex: 1;
    font-size: 0.75rem;
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    color: var(--text-primary);
    word-break: break-all;
  }

  .copy-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border-radius: 4px;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    flex-shrink: 0;
    transition:
      background 0.15s,
      color 0.15s;
  }

  .copy-btn:hover {
    background: var(--border);
    color: var(--text-primary);
  }

  .copy-btn :global(.copy-icon) {
    width: 0.875rem;
    height: 0.875rem;
  }

  .copy-btn :global(.copy-ok) {
    color: #10b981;
  }

  .npm-config-hint {
    margin-top: 0.5rem;
    font-size: 0.7rem;
    color: var(--text-secondary);
  }

  .hint-label {
    font-weight: 600;
  }

  .hint-code {
    display: block;
    margin-top: 0.25rem;
    padding: 0.375rem 0.625rem;
    border-radius: 4px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.7rem;
    word-break: break-all;
    color: var(--text-primary);
  }

  /* Key list */
  .keys-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .key-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
  }

  .key-info {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .key-name {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .key-prefix {
    font-size: 0.7rem;
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    color: var(--text-secondary);
  }

  .key-meta {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-shrink: 0;
  }

  .key-date {
    font-size: 0.7rem;
    color: var(--text-secondary);
  }

  .delete-key-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border-radius: 4px;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    transition:
      background 0.15s,
      color 0.15s;
  }

  .delete-key-btn:hover:not(:disabled) {
    background: rgba(220, 38, 38, 0.1);
    color: #dc2626;
  }

  .delete-key-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .delete-key-btn :global(.delete-icon) {
    width: 0.875rem;
    height: 0.875rem;
  }

  .delete-key-btn :global(.delete-icon-spin) {
    animation: spin 1s linear infinite;
  }

  .empty-keys {
    font-size: 0.8rem;
    color: var(--text-secondary);
    text-align: center;
    padding: 1rem 0;
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
