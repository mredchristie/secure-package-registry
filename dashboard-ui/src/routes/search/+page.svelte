<script lang="ts">
  import { searchAPI } from "$lib/api";
  import type { PackageSummary, Ecosystem } from "$lib/types/api";

  function trustColor(score: number): string {
    if (score >= 90) return "trust-high";
    if (score >= 70) return "trust-medium";
    return "trust-low";
  }

  let searchQuery = $state("");
  let selectedEcosystem = $state<"" | Ecosystem>("");
  let searchResults = $state<PackageSummary[]>([]);
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

  const filteredResults = $derived(
    searchResults.filter((pkg) => {
      const query = searchQuery.trim().toLowerCase();

      const matchesQuery =
        !query ||
        pkg.identifier.toLowerCase().includes(query) ||
        (pkg.description ?? "").toLowerCase().includes(query) ||
        (pkg.tags ?? []).some((tag) => tag.toLowerCase().includes(query));

      const matchesEcosystem =
        !selectedEcosystem || pkg.ecosystem === selectedEcosystem;

      return matchesQuery && matchesEcosystem;
    }),
  );

  async function searchPackages() {
    if (!searchQuery.trim() && !selectedEcosystem) {
      error = "Please enter a search term or select an ecosystem.";
      hasSearched = false;
      searchResults = [];
      return;
    }

    loading = true;
    error = "";
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

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Enter") {
      searchPackages();
    }
  }
</script>

<main>
  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  <div class="search-card">
    <div class="search-row">
      <input
        type="text"
        bind:value={searchQuery}
        onkeydown={handleKeydown}
        placeholder="Search packages…"
        class="search-input"
      />

      <select bind:value={selectedEcosystem} class="eco-select">
        {#each ecosystems as ecosystem}
          <option value={ecosystem.value}>{ecosystem.label}</option>
        {/each}
      </select>

      <button onclick={searchPackages} disabled={loading} class="btn-search">
        {loading ? "Searching…" : "Search"}
      </button>

      <button class="btn-add" type="button">+ Add Package</button>
    </div>

    {#if hasSearched && !loading}
      <div class="results-meta">
        {filteredResults.length} package{filteredResults.length !== 1
          ? "s"
          : ""} found
      </div>
    {/if}
  </div>

  {#if loading}
    <div class="state-msg">Searching…</div>
  {:else if hasSearched && filteredResults.length === 0}
    <div class="state-msg">No packages matched your query.</div>
  {:else if filteredResults.length > 0}
    <ul class="cards">
      {#each filteredResults as pkg}
        <li>
          <a
            class="card-link"
            href={`/detail-page/${encodeURIComponent(pkg.identifier)}?ecosystem=${pkg.ecosystem}&version=${pkg.latest_version}`}
          >
            <article class="card">
              <div class="card-header">
                <div class="card-id-row">
                  <span class="card-identifier">{pkg.identifier}</span>
                  <span class="eco-badge {pkg.ecosystem}">{pkg.ecosystem}</span>
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
                  {#each pkg.tags as tag}
                    <span class="result-tag">{tag}</span>
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
            </article>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</main>

<style>
  main {
    max-width: 1000px;
    margin: 0 auto;
    padding: 1.5rem 1.25rem 4rem;
  }

  .error-banner {
    margin-bottom: 1rem;
    padding: 1rem;
    border-radius: 12px;
    border: 1px solid rgba(220, 38, 38, 0.2);
    background: rgba(220, 38, 38, 0.08);
    color: #dc2626;
  }

  .search-card {
    margin-bottom: 1rem;
  }

  .search-row {
    display: grid;
    grid-template-columns: 1fr 180px auto auto;
    gap: 0.625rem;
    align-items: center;
  }

  .search-input,
  .eco-select {
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

  .search-input:focus,
  .eco-select:focus {
    border-color: var(--accent);
  }

  .btn-search,
  .btn-add {
    padding: 0.625rem 1rem;
    font-size: 0.9rem;
    font-weight: 700;
    border-radius: 10px;
    cursor: pointer;
    white-space: nowrap;
  }

  .btn-search {
    border: 1px solid var(--accent);
    background: var(--accent);
    color: white;
  }

  .btn-search:hover:not(:disabled) {
    background: var(--accent-hover);
    border-color: var(--accent-hover);
  }

  .btn-search:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .btn-add {
    border: 1px solid var(--accent);
    background: transparent;
    color: var(--accent);
  }

  .btn-add:hover {
    background: rgba(29, 78, 216, 0.08);
  }

  .results-meta {
    margin-top: 0.75rem;
    font-size: 0.88rem;
    color: var(--text-secondary);
    font-weight: 600;
  }

  .state-msg {
    text-align: center;
    color: var(--text-secondary);
    padding: 3rem 0;
    font-size: 0.95rem;
  }

  .cards {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding: 0;
    margin: 0;
  }

  .card-link {
    display: block;
    text-decoration: none;
    color: inherit;
  }

  .card {
    border-radius: 14px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    padding: 1.5rem;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.5rem;
  }

  .card-id-row {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    min-width: 0;
  }

  .card-identifier {
    font-weight: 800;
    font-size: 1.05rem;
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

  .eco-badge.npm {
    border-color: rgba(252, 165, 165, 0.4);
    color: #b91c1c;
    background: #fef2f2;
  }

  .eco-badge.go {
    border-color: rgba(103, 232, 249, 0.4);
    color: #0e7490;
    background: #ecfeff;
  }

  .eco-badge.cargo {
    border-color: rgba(253, 186, 116, 0.4);
    color: #c2410c;
    background: #fff7ed;
  }

  .eco-badge.pypi {
    border-color: rgba(147, 197, 253, 0.4);
    color: #1d4ed8;
    background: #eff6ff;
  }

  .trust-pill {
    flex-shrink: 0;
    min-width: 2.8rem;
    padding: 0.125rem 0.625rem;
    font-size: 0.875rem;
    font-weight: 800;
    border-radius: 999px;
    border: 1.5px solid;
    text-align: center;
    background: transparent;
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

  @media (max-width: 640px) {
    .search-row {
      grid-template-columns: 1fr;
    }
  }
</style>
