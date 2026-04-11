<script lang="ts">
  import { packagesAPI } from "$lib/api";
  import type { Ecosystem, Package } from "$lib/types/api";
  import { Plus, AlertCircle, Loader2 } from "lucide-svelte";

  const ecosystems: Ecosystem[] = ["npm", "go", "cargo", "pypi"];

  const ecosystemColors: Record<Ecosystem, string> = {
    npm: "eco-npm",
    go: "eco-go",
    cargo: "eco-cargo",
    pypi: "eco-pypi",
  };

  let selectedEcosystem = $state<Ecosystem>("npm");
  let packages = $state<Package[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  // Add dialog state
  let addDialogOpen = $state(false);
  let newPackageName = $state("");
  let addError = $state<string | null>(null);
  let addLoading = $state(false);
  let addSuccess = $state<string | null>(null);

  async function loadPackages() {
    loading = true;
    error = null;
    try {
      const response = await packagesAPI.list(selectedEcosystem);
      packages = response.items;
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load packages";
    } finally {
      loading = false;
    }
  }

  async function handleAddPackage() {
    if (!newPackageName.trim()) {
      addError = "Package name is required";
      return;
    }

    addLoading = true;
    addError = null;
    addSuccess = null;

    try {
      const response = await packagesAPI.add({
        identifier: newPackageName.trim(),
        ecosystem: selectedEcosystem,
      });

      if (response.already_exists) {
        addSuccess = `Package "${response.identifier}" is already tracked`;
      } else {
        addSuccess = `Package "${response.identifier}" added successfully`;
        packages = [
          ...packages,
          {
            id: response.id,
            identifier: response.identifier,
            ecosystem: response.ecosystem,
          },
        ];
      }

      newPackageName = "";
      setTimeout(() => {
        addDialogOpen = false;
        addSuccess = null;
      }, 1500);
    } catch (e) {
      addError = e instanceof Error ? e.message : "Failed to add package";
    } finally {
      addLoading = false;
    }
  }

  function openAddDialog() {
    addDialogOpen = true;
    addError = null;
    addSuccess = null;
    newPackageName = "";
  }

  $effect(() => {
    loadPackages();
  });
</script>

<div class="page">
  <div class="page-header">
    <div>
      <h1 class="page-title">Packages</h1>
      <p class="page-subtitle">Manage tracked packages</p>
    </div>
    <button onclick={openAddDialog} class="add-button">
      <Plus class="icon-sm" />
      Add Package
    </button>
  </div>

  <div class="filter-bar">
    <select
      bind:value={selectedEcosystem}
      onchange={loadPackages}
      class="ecosystem-select"
    >
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
  {:else if packages.length === 0}
    <div class="empty-state">
      <p class="empty-text">
        No packages found for {selectedEcosystem}
      </p>
      <p class="empty-hint">Add a package to start tracking</p>
    </div>
  {:else}
    <div class="packages-grid">
      {#each packages as pkg}
        {@const eco = (pkg.ecosystem as Ecosystem) ?? selectedEcosystem}
        {@const colorClass = ecosystemColors[eco] ?? ecosystemColors.npm}
        <a
          href="/admin/packages/{pkg.ecosystem}/{encodeURIComponent(
            pkg.identifier,
          )}"
          class="package-card"
        >
          <div class="card-row">
            <div class="card-left">
              <span class="eco-badge {colorClass}">
                {eco}
              </span>
              <span class="package-name">
                {pkg.identifier}
              </span>
            </div>
            <div class="card-right">
              {#if pkg.latest_version}
                <span class="version-text">
                  v{pkg.latest_version}
                </span>
              {/if}
              <span class="view-link"> View → </span>
            </div>
          </div>
        </a>
      {/each}
    </div>
  {/if}
</div>

<!-- Add Package Dialog -->
{#if addDialogOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="dialog-overlay" onclick={() => (addDialogOpen = false)}>
    <div class="dialog-panel" onclick={(e) => e.stopPropagation()}>
      <div class="dialog-header">
        <h2 class="dialog-title">Add Package</h2>
        <p class="dialog-subtitle">
          Add a {selectedEcosystem} package to track
        </p>
      </div>

      <div class="dialog-body">
        <div class="form-field">
          <label class="form-label" for="packageName"> Package Name </label>
          <input
            id="packageName"
            type="text"
            bind:value={newPackageName}
            placeholder="e.g., express"
            onkeydown={(e: KeyboardEvent) =>
              e.key === "Enter" && handleAddPackage()}
            class="form-input"
          />
        </div>

        {#if addError}
          <div class="alert alert-error">
            {addError}
          </div>
        {/if}

        {#if addSuccess}
          <div class="alert alert-success">
            {addSuccess}
          </div>
        {/if}
      </div>

      <div class="dialog-footer">
        <button onclick={() => (addDialogOpen = false)} class="btn-secondary">
          Cancel
        </button>
        <button
          onclick={handleAddPackage}
          disabled={addLoading}
          class="btn-primary"
        >
          {#if addLoading}
            <Loader2 class="icon-spin" />
          {/if}
          Add Package
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .page {
    padding: 2rem;
  }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
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

  .add-button {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: #fff;
    background: var(--accent);
    border: none;
    border-radius: 6px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    cursor: pointer;
    transition: background 0.15s;
  }

  .add-button:hover {
    background: var(--accent-hover);
  }

  .add-button :global(.icon-sm) {
    width: 1rem;
    height: 1rem;
  }

  .filter-bar {
    margin-bottom: 1.5rem;
  }

  .ecosystem-select {
    height: 2.5rem;
    width: 12rem;
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

  .packages-grid {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .package-card {
    display: block;
    text-decoration: none;
    border-radius: 12px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    padding: 1rem;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    transition:
      transform 0.15s,
      border-color 0.15s,
      box-shadow 0.15s;
  }

  .package-card:hover {
    transform: translateY(-2px);
    border-color: rgba(29, 78, 216, 0.35);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  }

  .card-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .card-left {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-width: 0;
  }

  .eco-badge {
    display: inline-block;
    flex-shrink: 0;
    padding: 0.125rem 0.625rem;
    font-size: 0.7rem;
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
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .package-card:hover .package-name {
    color: var(--accent);
  }

  .card-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-shrink: 0;
  }

  .version-text {
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .view-link {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    opacity: 0.6;
    transition: opacity 0.15s;
  }

  .package-card:hover .view-link {
    opacity: 1;
    color: var(--text-primary);
  }

  /* Dialog */
  .dialog-overlay {
    position: fixed;
    inset: 0;
    z-index: 50;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .dialog-panel {
    width: 100%;
    max-width: 28rem;
    border-radius: 8px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    padding: 1.5rem;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
  }

  .dialog-header {
    margin-bottom: 1rem;
  }

  .dialog-title {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .dialog-subtitle {
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .dialog-body {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .form-field {
    display: flex;
    flex-direction: column;
  }

  .form-label {
    margin-bottom: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .form-input {
    height: 2.5rem;
    width: 100%;
    padding: 0 0.75rem;
    font-size: 0.875rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    outline: none;
    transition: border-color 0.15s;
  }

  .form-input::placeholder {
    color: var(--text-secondary);
  }

  .form-input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px rgba(29, 78, 216, 0.15);
  }

  .alert {
    padding: 0.75rem;
    border-radius: 6px;
    font-size: 0.875rem;
  }

  .alert-error {
    background: rgba(220, 38, 38, 0.08);
    color: #dc2626;
  }

  .alert-success {
    background: rgba(22, 163, 74, 0.08);
    color: #16a34a;
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1.5rem;
  }

  .btn-secondary {
    padding: 0.5rem 1rem;
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

  .btn-secondary:hover {
    background: var(--bg-primary);
  }

  .btn-primary {
    display: inline-flex;
    align-items: center;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: #fff;
    background: var(--accent);
    border: none;
    border-radius: 6px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-primary:hover {
    background: var(--accent-hover);
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary :global(.icon-spin) {
    width: 1rem;
    height: 1rem;
    margin-right: 0.5rem;
    animation: spin 1s linear infinite;
  }
</style>
