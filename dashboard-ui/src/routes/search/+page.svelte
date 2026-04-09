<script lang="ts">
  import { searchAPI } from "$lib/api";
  import type { PackageSummary, Ecosystem } from "$lib/types/api";
  import { page } from "$app/state";

  // Initialize from URL params
  const initialQ = page.url.searchParams.get("q") ?? "";
  const initialEcosystem =
    (page.url.searchParams.get("ecosystem") as Ecosystem | "") ?? "";
  const initialPage = parseInt(page.url.searchParams.get("page") ?? "1", 10);

  let searchQuery = $state(initialQ);
  let selectedEcosystem = $state<"" | Ecosystem>(initialEcosystem);
  let searchResults = $state<PackageSummary[]>([]);
  let totalCount = $state(0);
  let currentPage = $state(initialPage);
  let pageSize = $state(20);
  let loading = $state(false);
  let error = $state("");
  let hasSearched = $state(false);
  let debounceTimer = $state<ReturnType<typeof setTimeout> | null>(null);

  const ecosystems = [
    { label: "All Ecosystems", value: "" as const },
    { label: "npm", value: "npm" as const },
    { label: "Go", value: "go" as const },
    { label: "Cargo", value: "cargo" as const },
    { label: "PyPI", value: "pypi" as const },
  ];

  const totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));

  function updateUrlParams() {
    const params = new URLSearchParams();
    if (searchQuery.trim()) params.set("q", searchQuery.trim());
    if (selectedEcosystem) params.set("ecosystem", selectedEcosystem);
    if (currentPage > 1) params.set("page", currentPage.toString());

    const newUrl = `${window.location.pathname}${params.toString() ? "?" + params.toString() : ""}`;
    window.history.replaceState({}, "", newUrl);
  }

  async function searchPackages(resetPage = false) {
    if (resetPage) currentPage = 1;
    loading = true;
    error = "";
    hasSearched = true;
    updateUrlParams();

    try {
      const data = await searchAPI.search(
        searchQuery.trim(),
        selectedEcosystem || undefined,
        currentPage,
        pageSize,
      );
      searchResults = data.items ?? [];
      totalCount = data.total_count;
    } catch {
      error = "Failed to search packages. Please try again.";
      searchResults = [];
      totalCount = 0;
    } finally {
      loading = false;
    }
  }

  function handleInput() {
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => searchPackages(true), 300);
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Enter") {
      if (debounceTimer) clearTimeout(debounceTimer);
      searchPackages(true);
    }
  }

  function handleEcosystemChange() {
    searchPackages(true);
  }

  function goToPage(p: number) {
    currentPage = p;
    searchPackages();
  }

  function packageUrl(pkg: PackageSummary): string {
    return `/packages/${pkg.ecosystem}/${encodeURIComponent(pkg.identifier)}/${pkg.latest_version}`;
  }

  // Auto-search on mount if URL has query params, otherwise show all
  $effect(() => {
    if (!hasSearched) {
      searchPackages();
    }
  });
</script>

<main>
  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  <div class="search-bar">
    <input
      type="text"
      bind:value={searchQuery}
      oninput={handleInput}
      onkeydown={handleKeydown}
      placeholder="Search packages…"
      class="search-input"
    />
    <select
      bind:value={selectedEcosystem}
      onchange={handleEcosystemChange}
      class="eco-select"
    >
      {#each ecosystems as ecosystem}
        <option value={ecosystem.value}>{ecosystem.label}</option>
      {/each}
    </select>
  </div>

  {#if hasSearched && !loading}
    <div class="results-meta">
      {totalCount} package{totalCount !== 1 ? "s" : ""}
      {#if searchQuery.trim()}
        matching "{searchQuery.trim()}"
      {/if}
    </div>
  {/if}

  {#if loading}
    <div class="state-msg">Searching…</div>
  {:else if hasSearched && searchResults.length === 0}
    <div class="state-msg">No packages matched your query.</div>
  {:else if searchResults.length > 0}
    <ul class="results">
      {#each searchResults as pkg}
        <li>
          <a class="result-link" href={packageUrl(pkg)}>
            <div class="result-row">
              <span class="result-name">{pkg.identifier}</span>
              <span class="eco-badge {pkg.ecosystem}">{pkg.ecosystem}</span>
              <span class="version-text">v{pkg.latest_version}</span>
            </div>
          </a>
        </li>
      {/each}
    </ul>

    {#if totalPages > 1}
      <nav class="pagination">
        <button
          onclick={() => goToPage(currentPage - 1)}
          disabled={currentPage <= 1}
          class="page-btn"
        >
          Previous
        </button>
        <span class="page-info">
          Page {currentPage} of {totalPages}
        </span>
        <button
          onclick={() => goToPage(currentPage + 1)}
          disabled={currentPage >= totalPages}
          class="page-btn"
        >
          Next
        </button>
      </nav>
    {/if}
  {/if}
</main>

<style>
  main {
    max-width: 800px;
    margin: 0 auto;
    padding: 1.5rem 1.25rem 4rem;
  }

  .error-banner {
    margin-bottom: 1rem;
    padding: 0.75rem 1rem;
    border-radius: 8px;
    border: 1px solid rgba(220, 38, 38, 0.2);
    background: rgba(220, 38, 38, 0.08);
    color: #dc2626;
    font-size: 0.875rem;
  }

  .search-bar {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .search-input {
    flex: 1;
    padding: 0.625rem 0.875rem;
    font-size: 0.9rem;
    border-radius: 8px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    outline: none;
    transition: border-color 0.15s;
  }

  .search-input::placeholder {
    color: var(--text-secondary);
  }

  .search-input:focus {
    border-color: var(--accent);
  }

  .eco-select {
    padding: 0.625rem 0.75rem;
    font-size: 0.875rem;
    border-radius: 8px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    outline: none;
    cursor: pointer;
  }

  .eco-select:focus {
    border-color: var(--accent);
  }

  .results-meta {
    font-size: 0.8rem;
    color: var(--text-secondary);
    font-weight: 600;
    margin-bottom: 0.75rem;
  }

  .state-msg {
    text-align: center;
    color: var(--text-secondary);
    padding: 3rem 0;
    font-size: 0.9rem;
  }

  .results {
    list-style: none;
    padding: 0;
    margin: 0;
    border: 1px solid var(--card-border);
    border-radius: 10px;
    background: var(--card-bg);
    overflow: hidden;
  }

  .results li + li {
    border-top: 1px solid var(--border);
  }

  .result-link {
    display: block;
    text-decoration: none;
    color: inherit;
    padding: 0.75rem 1rem;
    transition: background 0.1s;
  }

  .result-link:hover {
    background: var(--bg-secondary);
  }

  .result-row {
    display: flex;
    align-items: center;
    gap: 0.625rem;
  }

  .result-name {
    font-weight: 700;
    font-size: 0.95rem;
    color: var(--text-primary);
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .eco-badge {
    display: inline-block;
    padding: 0.1rem 0.4rem;
    font-size: 0.65rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.02em;
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

  .version-text {
    margin-left: auto;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-secondary);
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    flex-shrink: 0;
  }

  .pagination {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    margin-top: 1rem;
  }

  .page-btn {
    padding: 0.4rem 0.85rem;
    font-size: 0.8rem;
    font-weight: 600;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    cursor: pointer;
    transition:
      background 0.15s,
      border-color 0.15s;
  }

  .page-btn:hover:not(:disabled) {
    border-color: var(--accent);
    color: var(--accent);
  }

  .page-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .page-info {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-secondary);
  }

  @media (max-width: 640px) {
    .search-bar {
      flex-direction: column;
    }
  }
</style>
