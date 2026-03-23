<script lang="ts">
  import { searchAPI } from "$lib/api";
  import type {
    PackageSummary,
    PackageVersionDetail,
    Ecosystem,
  } from "$lib/types/api";

  function decodeTagValue(data: string): string {
    try {
      const parsed = JSON.parse(atob(data));
      if (parsed === null) return "null";
      if (typeof parsed === "object") return JSON.stringify(parsed);
      return String(parsed);
    } catch {
      return data;
    }
  }

  function trustColor(score: number): string {
    if (score >= 90) return "trust-high";
    if (score >= 70) return "trust-medium";
    return "trust-low";
  }

  const ecosystemColors: Record<Ecosystem, string> = {
    npm: "eco-npm",
    go: "eco-go",
    cargo: "eco-cargo",
    pypi: "eco-pypi",
  };

  let searchQuery = $state("");
  let selectedEcosystem = $state<"" | Ecosystem>("");
  let searchResults = $state<PackageSummary[]>([]);
  let selectedPackage = $state<PackageVersionDetail | null>(null);
  let loading = $state(false);
  let error = $state("");
  let hasSearched = $state(false);

  const ecosystems = [
    { label: "All Ecosystems", value: "" as const },
    { label: "npm", value: "npm" as const },
    { label: "Go", value: "go" as const },
    { label: "Cargo", value: "cargo" as const },
    { label: "PyPI", value: "pypi" as const },
  ];

  let filteredResults = $derived(
    searchResults.filter((p) => {
      const q = searchQuery.trim().toLowerCase();

      const matchesQuery =
        !q ||
        p.identifier.toLowerCase().includes(q) ||
        (p.description ?? "").toLowerCase().includes(q) ||
        (p.tags ?? []).some((t) => t.toLowerCase().includes(q));

      const matchesEco =
        !selectedEcosystem || p.ecosystem === selectedEcosystem;

      return matchesQuery && matchesEco;
    }),
  );

  async function searchPackages() {
    if (!searchQuery.trim()) {
      error = "Please enter a search term.";
      return;
    }

    loading = true;
    error = "";
    selectedPackage = null;
    hasSearched = true;

    try {
      const data = await searchAPI.search(
        searchQuery.trim(),
        selectedEcosystem || undefined,
      );
      searchResults = data.items ?? [];
    } catch {
      error = "Failed to search packages. Please try again.";
      searchResults = [];
    } finally {
      loading = false;
    }
  }

  async function selectPackage(pkg: PackageSummary) {
    loading = true;
    error = "";

    try {
      selectedPackage = await searchAPI.getVersion(
        pkg.ecosystem,
        pkg.identifier,
        pkg.latest_version,
      );
    } catch {
      error = "Failed to fetch package details.";
      selectedPackage = null;
    } finally {
      loading = false;
    }
  }

  function clearSelection() {
    selectedPackage = null;
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Enter") searchPackages();
  }
</script>

<main>
  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  {#if selectedPackage}
    <!-- Detail View -->
    <div class="card">
      <button onclick={clearSelection} class="back-button">
        ← Back to results
      </button>

      <div class="detail-header">
        <div>
          <h2>{selectedPackage.identifier}</h2>
          <span
            class="eco-badge {ecosystemColors[
              selectedPackage.ecosystem as Ecosystem
            ] ?? 'eco-npm'}"
          >
            {selectedPackage.ecosystem}
          </span>
        </div>

        <div class="trust-circle {trustColor(selectedPackage.trust_level)}">
          <span class="trust-score">{selectedPackage.trust_level}</span>
          <span class="trust-label">Trust</span>
        </div>
      </div>

      <div class="detail-grid">
        <div class="detail-item">
          <div class="detail-label">Version</div>
          <div class="detail-value">
            {selectedPackage.version}
            {#if selectedPackage.latest}
              <span class="badge-latest">Latest</span>
            {/if}
          </div>
        </div>

        <div class="detail-item">
          <div class="detail-label">Trust Level</div>
          <div class="detail-value">{selectedPackage.trust_level} / 100</div>
        </div>

        <div class="detail-item">
          <div class="detail-label">Source Tag</div>
          <div class="detail-value mono">{selectedPackage.source.tag}</div>
        </div>

        <div class="detail-item">
          <div class="detail-label">Commit</div>
          <div class="detail-value mono">{selectedPackage.source.commit}</div>
        </div>

        {#if selectedPackage.maintainer_notes}
          <div class="detail-item full-width">
            <div class="detail-label">Maintainer Notes</div>
            <p class="notes">{selectedPackage.maintainer_notes}</p>
          </div>
        {/if}

        {#if selectedPackage.tags?.length}
          <div class="detail-item full-width">
            <div class="detail-label">Tags</div>
            <div class="tags-list">
              {#each selectedPackage.tags as tag}
                <div class="tag-item">
                  <span class="tag-label">{tag.label}</span>
                  <span class="tag-value">{decodeTagValue(tag.data)}</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    </div>
  {:else}
    <!-- Search View -->
    <div class="card search-card">
      <div class="search-row">
        <input
          type="text"
          bind:value={searchQuery}
          onkeydown={handleKeydown}
          placeholder="Search packages..."
          class="search-input"
        />

        <select bind:value={selectedEcosystem} class="ecosystem-select">
          {#each ecosystems as eco}
            <option value={eco.value}>{eco.label}</option>
          {/each}
        </select>

        <button
          onclick={searchPackages}
          disabled={loading}
          class="search-button"
        >
          {loading ? "Searching..." : "Search"}
        </button>
      </div>

      {#if hasSearched && !loading}
        <div class="result-count">
          {searchResults.length} package{searchResults.length !== 1 ? "s" : ""} found
        </div>
      {/if}
    </div>

    {#if loading}
      <div class="status-message">Searching...</div>
    {:else if hasSearched && searchResults.length === 0}
      <div class="status-message">No packages matched your query.</div>
    {:else if searchResults.length > 0}
      <ul class="results-list">
        {#each filteredResults as pkg}
          <li>
            <button
              class="result-card"
              onclick={() => selectPackage(pkg)}
              type="button"
            >
              <div class="result-top">
                <div class="result-name-row">
                  <span class="result-name">{pkg.identifier}</span>
                  <span
                    class="eco-badge {ecosystemColors[pkg.ecosystem] ??
                      'eco-npm'}"
                  >
                    {pkg.ecosystem}
                  </span>
                </div>

                <span class="trust-pill {trustColor(pkg.trustScore ?? 0)}">
                  {pkg.trustScore ?? 0}
                </span>
              </div>

              {#if pkg.description}
                <p class="result-description">{pkg.description}</p>
              {/if}

              {#if pkg.tags?.length}
                <div class="result-tags">
                  {#each pkg.tags as t}
                    <span class="result-tag">{t}</span>
                  {/each}
                </div>
              {/if}

              <div class="result-meta">
                <span>v{pkg.latest_version}</span>
                {#if pkg.author}
                  <span>{pkg.author}</span>
                {/if}
                {#if pkg.updatedAgo}
                  <span>{pkg.updatedAgo}</span>
                {/if}
                {#if pkg.tier}
                  <span class="result-tier">{pkg.tier}</span>
                {/if}
              </div>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  {/if}
</main>

<style>
  main {
    max-width: 1000px;
    margin: 0 auto;
    padding: 1.5rem 1.25rem 4rem;
  }

  /* Error banner */
  .error-banner {
    margin-bottom: 1rem;
    padding: 1rem;
    border-radius: 12px;
    border: 1px solid rgba(220, 38, 38, 0.2);
    background: rgba(220, 38, 38, 0.08);
    color: #dc2626;
  }

  /* Card base */
  .card {
    border-radius: 14px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    padding: 1.5rem;
  }

  /* Search form */
  .search-card {
    margin-bottom: 1rem;
  }

  .search-row {
    display: grid;
    grid-template-columns: 1fr 180px auto;
    gap: 0.625rem;
    align-items: center;
  }

  .search-input {
    padding: 0.625rem 0.875rem;
    font-size: 0.93rem;
    border-radius: 10px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    outline: none;
    transition: border-color 0.2s;
  }

  .search-input::placeholder {
    color: var(--text-secondary);
  }

  .search-input:focus {
    border-color: var(--accent);
  }

  .ecosystem-select {
    padding: 0.625rem 0.875rem;
    font-size: 0.93rem;
    border-radius: 10px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    outline: none;
    cursor: pointer;
    appearance: auto;
  }

  .ecosystem-select:focus {
    border-color: var(--accent);
  }

  .search-button {
    padding: 0.625rem 1rem;
    font-size: 0.875rem;
    font-weight: 700;
    white-space: nowrap;
    border-radius: 10px;
    border: 1px solid var(--accent);
    background: var(--accent);
    color: #fff;
    cursor: pointer;
    transition:
      background 0.2s,
      border-color 0.2s;
  }

  .search-button:hover:not(:disabled) {
    background: var(--accent-hover);
    border-color: var(--accent-hover);
  }

  .search-button:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .result-count {
    margin-top: 0.75rem;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .status-message {
    padding: 3rem 0;
    text-align: center;
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  /* Results list */
  .results-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding: 0;
  }

  .result-card {
    display: block;
    width: 100%;
    text-align: left;
    padding: 1rem;
    border-radius: 12px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    cursor: pointer;
    transition:
      transform 0.15s,
      border-color 0.15s;
  }

  .result-card:hover {
    transform: translateY(-1px);
    border-color: rgba(29, 78, 216, 0.35);
  }

  .result-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    margin-bottom: 0.25rem;
  }

  .result-name-row {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    min-width: 0;
  }

  .result-name {
    font-size: 1.05rem;
    font-weight: 800;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .result-description {
    margin: 0 0 0.625rem;
    font-size: 0.875rem;
    line-height: 1.625;
    color: var(--text-secondary);
  }

  .result-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.375rem;
    margin-bottom: 0.75rem;
  }

  .result-tag {
    padding: 0.125rem 0.625rem;
    font-size: 0.76rem;
    font-weight: 700;
    border-radius: 999px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-secondary);
  }

  .result-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    padding-top: 0.625rem;
    border-top: 1px solid var(--border);
    font-size: 0.82rem;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .result-tier {
    margin-left: auto;
    font-weight: 700;
    color: var(--accent);
  }

  /* Ecosystem badges */
  .eco-badge {
    display: inline-block;
    padding: 0.125rem 0.5rem;
    font-size: 0.7rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.025em;
    border-radius: 999px;
    border: 1px solid;
    flex-shrink: 0;
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

  /* Trust indicators */
  .trust-pill {
    flex-shrink: 0;
    padding: 0.125rem 0.625rem;
    font-size: 0.875rem;
    font-weight: 800;
    border-radius: 999px;
    border: 1.5px solid;
    text-align: center;
  }

  .trust-circle {
    flex-shrink: 0;
    width: 68px;
    height: 68px;
    border-radius: 50%;
    border: 3px solid;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }

  .trust-score {
    font-size: 1.125rem;
    font-weight: 900;
    line-height: 1;
  }

  .trust-label {
    font-size: 0.65rem;
    font-weight: 700;
    text-transform: uppercase;
    color: var(--text-secondary);
  }

  .trust-high {
    color: #16a34a;
    border-color: #22c55e;
  }

  .trust-medium {
    color: #f59e0b;
    border-color: #fbbf24;
  }

  .trust-low {
    color: #dc2626;
    border-color: #ef4444;
  }

  /* Detail view */
  .back-button {
    border: none;
    background: transparent;
    font-size: 0.875rem;
    font-weight: 700;
    color: var(--accent);
    cursor: pointer;
    padding: 0;
    margin-bottom: 1rem;
  }

  .back-button:hover {
    text-decoration: underline;
  }

  .detail-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1.25rem;
  }

  .detail-header h2 {
    font-size: 1.5rem;
    font-weight: 900;
    color: var(--text-primary);
    margin-bottom: 0.25rem;
  }

  .detail-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }

  .detail-item {
    padding: 0.75rem;
    border-radius: 10px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
  }

  .detail-item.full-width {
    grid-column: span 2;
  }

  .detail-label {
    margin-bottom: 0.25rem;
    font-size: 0.7rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .detail-value {
    font-weight: 600;
    color: var(--text-primary);
  }

  .mono {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.875rem;
  }

  .badge-latest {
    display: inline-block;
    margin-left: 0.25rem;
    padding: 0.125rem 0.375rem;
    font-size: 0.72rem;
    font-weight: 800;
    border-radius: 6px;
    background: #16a34a;
    color: #fff;
    vertical-align: middle;
  }

  .notes {
    margin: 0;
    font-size: 0.875rem;
    line-height: 1.625;
    color: var(--text-primary);
  }

  .tags-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .tag-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.25rem 0.625rem;
    border-radius: 8px;
    border: 1px solid var(--border);
    background: var(--bg-primary);
  }

  .tag-label {
    font-size: 0.875rem;
    font-weight: 800;
    color: var(--accent);
  }

  .tag-value {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-secondary);
  }

  /* Responsive */
  @media (max-width: 640px) {
    .search-row {
      grid-template-columns: 1fr;
    }

    .detail-grid {
      grid-template-columns: 1fr;
    }

    .detail-item.full-width {
      grid-column: span 1;
    }
  }
</style>
