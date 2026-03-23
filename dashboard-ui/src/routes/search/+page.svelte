<script lang="ts">
  import { onMount } from "svelte";
  // Types matching the API
  interface PackageSummary {
    identifier: string;
    ecosystem: "npm" | "go" | "cargo" | "pypi";
    latest_version: string;
    description: string;
    author: string;
    updatedAgo: string;
    trustScore: number;
    tier: string;
    tags: string[];
  }

  interface SearchResult {
    items: PackageSummary[];
  }

  interface PackageVersion {
    identifier: string;
    ecosystem: string;
    version: string;
    latest: boolean;
    source: {
      url: string;
      tag: string;
      commit: string;
    };
    trust_level: number;
    maintainer_notes: string;
    tags: Array<{
      label: string;
      value_type: "boolean" | "integer" | "float";
      data: string; // base64 encoded JSON
    }>;
  }

  // Decode base64 and parse JSON value
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
    if (score >= 90) return "#16a34a";
    if (score >= 70) return "#f59e0b";
    return "#dc2626";
  }

  let searchQuery = "";
  let selectedEcosystem: "" | "npm" | "go" | "cargo" | "pypi" = "";
  let searchResults: PackageSummary[] = [];
  let selectedPackage: PackageVersion | null = null;

  let loading = false;
  let error = "";
  let hasSearched = false;

  const ecosystems = [
    { label: "All Ecosystems", value: "" },
    { label: "npm", value: "npm" },
    { label: "Go", value: "go" },
    { label: "Cargo", value: "cargo" },
    { label: "PyPI", value: "pypi" },
  ] as const;

  onMount(async () => {
    await loadDefaultPackages();
  });

  async function loadDefaultPackages() {
    loading = true;
    error = "";

    try {
      const response = await fetch(`/api/v1/svc/packages`);
      if (!response.ok) throw new Error();

      const data: SearchResult = await response.json();
      searchResults = data.items ?? [];
      hasSearched = true;
    } catch {
      error = "Failed to load packages.";
    } finally {
      loading = false;
    }
  }

  $: filteredResults = searchResults.filter((p) => {
    const q = searchQuery.trim().toLowerCase();

    const matchesQuery =
      !q ||
      p.identifier.toLowerCase().includes(q) ||
      p.description.toLowerCase().includes(q) ||
      p.tags.some((t) => t.toLowerCase().includes(q));

    const matchesEco = !selectedEcosystem || p.ecosystem === selectedEcosystem;

    return matchesQuery && matchesEco;
  });

  async function searchPackages() {
    loading = true;
    error = "";
    selectedPackage = null;
    hasSearched = true;

    try {
      const params = new URLSearchParams();

      if (searchQuery.trim()) {
        params.append("q", searchQuery);
      }

      if (selectedEcosystem) {
        params.append("ecosystem", selectedEcosystem);
      }

      const queryString = params.toString();
      const url = queryString
        ? `/api/v1/svc/packages?${queryString}`
        : `/api/v1/svc/packages`;

      const response = await fetch(url);
      if (!response.ok) throw new Error("Search failed");

      const data: SearchResult = await response.json();
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
      const safeIdentifier = encodeURIComponent(pkg.identifier);

      const response = await fetch(
        `/api/v1/svc/packages/${pkg.ecosystem}/${safeIdentifier}/${pkg.latest_version}`,
      );
      if (!response.ok) throw new Error("Failed to fetch package details");

      selectedPackage = await response.json();
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
  // Handle Enter key in search input
  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Enter") searchPackages();
  }
</script>

<main>
  {#if error}
    <div class="error">{error}</div>
  {/if}

  {#if selectedPackage}
    <!-- ── DETAIL VIEW ── -->
    <div class="detail-card">
      <button class="back-button" on:click={clearSelection}
        >← Back to results</button
      >

      <div class="detail-header">
        <div>
          <h2>{selectedPackage.identifier}</h2>
          <span class="eco-badge {selectedPackage.ecosystem}">
            {selectedPackage.ecosystem}
          </span>
        </div>

        <div
          class="trust-circle"
          style="--tc: {trustColor(selectedPackage.trust_level)}"
        >
          <span class="trust-num">{selectedPackage.trust_level}</span>
          <span class="trust-lbl">Trust</span>
        </div>
      </div>

      <div class="detail-grid">
        <div class="detail-item">
          <div class="label">Version</div>
          <div class="value">
            {selectedPackage.version}
            {#if selectedPackage.latest}<span class="badge-latest">Latest</span
              >{/if}
          </div>
        </div>

        <div class="detail-item">
          <div class="label">Trust Level</div>
          <div class="value">{selectedPackage.trust_level} / 100</div>
        </div>

        <div class="detail-item">
          <div class="label">Source Tag</div>
          <div class="value mono">{selectedPackage.source.tag}</div>
        </div>

        <div class="detail-item">
          <div class="label">Commit</div>
          <div class="value mono">{selectedPackage.source.commit}</div>
        </div>

        <div class="detail-item full-width">
          <div class="label">Maintainer Notes</div>
          <p class="notes">{selectedPackage.maintainer_notes}</p>
        </div>

        {#if selectedPackage.tags?.length}
          <div class="detail-item full-width">
            <div class="label">Tags</div>
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
    <!-- ── SEARCH VIEW ── -->
    <div class="search-card">
      <div class="search-row">
        <input
          type="text"
          bind:value={searchQuery}
          on:keydown={handleKeydown}
          placeholder="Search packages…"
          class="search-input"
        />

        <select bind:value={selectedEcosystem} class="eco-select">
          {#each ecosystems as eco}
            <option value={eco.value}>{eco.label}</option>
          {/each}
        </select>

        <button on:click={searchPackages} disabled={loading} class="btn-search">
          {loading ? "Searching…" : "Search"}
        </button>

        <button class="btn-add" type="button">+ Add Package</button>
      </div>

      {#if hasSearched && !loading}
        <div class="results-meta">
          {searchResults.length} package{searchResults.length !== 1 ? "s" : ""} found
        </div>
      {/if}
    </div>

    {#if loading}
      <div class="state-msg">Searching…</div>
    {:else if hasSearched && searchResults.length === 0}
      <div class="state-msg">No packages matched your query.</div>
    {:else if searchResults.length > 0}
      <ul class="cards">
        {#each filteredResults as pkg}
          <li>
            <button
              class="card"
              on:click={() => selectPackage(pkg)}
              type="button"
            >
              <div class="card-header">
                <div class="card-id-row">
                  <span class="card-identifier">{pkg.identifier}</span>
                  <span class="eco-badge {pkg.ecosystem}">{pkg.ecosystem}</span>
                </div>

                <div
                  class="trust-pill"
                  style="--tc: {trustColor(pkg.trustScore ?? 0)}"
                >
                  {pkg.trustScore ?? 0}
                </div>
              </div>

              <p class="card-desc">{pkg.description}</p>

              <div class="card-tags">
                {#each pkg.tags as t}
                  <span class="tag">{t}</span>
                {/each}
              </div>

              <div class="card-footer">
                <span>v{pkg.latest_version}</span>
                <span>{pkg.author}</span>
                <span>{pkg.updatedAgo}</span>
                <span class="tier-label">{pkg.tier}</span>
              </div>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  {/if}
</main>

<footer>SPR &copy; 2026</footer>

<style>
  main {
    max-width: 1000px;
    margin: 0 auto;
    padding: 1.5rem 1.25rem 4rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    color: var(--text-primary);
  }

  .search-card {
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    border-radius: 14px;
    padding: 1.1rem;
  }

  .search-row {
    display: grid;
    grid-template-columns: 1fr 180px auto auto;
    gap: 0.65rem;
    align-items: center;
  }

  .search-input,
  .eco-select {
    padding: 0.7rem 0.9rem;
    border-radius: 10px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    font-size: 0.93rem;
    outline: none;
    transition: border-color 0.15s;
    min-width: 0;
  }

  .search-input:focus,
  .eco-select:focus {
    border-color: var(--accent);
  }

  .btn-search,
  .btn-add {
    padding: 0.7rem 1.1rem;
    font-size: 0.9rem;
    font-weight: 700;
    border-radius: 10px;
    cursor: pointer;
    white-space: nowrap;
    transition:
      background 0.15s,
      border-color 0.15s;
  }

  .btn-search {
    background: var(--accent);
    border: 1px solid var(--accent);
    color: #ffffff;
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
    background: transparent;
    border: 1px solid var(--accent);
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

  .error {
    background: rgba(220, 38, 38, 0.08);
    border: 1px solid rgba(220, 38, 38, 0.2);
    color: #b91c1c;
    padding: 0.9rem 1rem;
    border-radius: 12px;
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

  .card {
    width: 100%;
    text-align: left;
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    border-radius: 12px;
    padding: 1rem 1.1rem;
    cursor: pointer;
    transition:
      border-color 0.15s,
      transform 0.1s;
  }

  .card:hover {
    border-color: rgba(29, 78, 216, 0.35);
    transform: translateY(-1px);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.45rem;
    gap: 0.75rem;
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

  .trust-pill {
    font-weight: 800;
    font-size: 0.88rem;
    color: var(--tc);
    border: 1.5px solid var(--tc);
    border-radius: 999px;
    padding: 0.2rem 0.6rem;
    min-width: 2.8rem;
    text-align: center;
    background: transparent;
  }

  .card-desc {
    color: var(--text-secondary);
    font-size: 0.9rem;
    line-height: 1.5;
    margin: 0 0 0.65rem 0;
  }

  .card-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-bottom: 0.75rem;
  }

  .tag {
    font-size: 0.76rem;
    font-weight: 700;
    padding: 0.25rem 0.6rem;
    border-radius: 999px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-secondary);
  }

  .card-footer {
    display: flex;
    gap: 1.2rem;
    font-size: 0.82rem;
    color: var(--text-secondary);
    border-top: 1px solid var(--border);
    padding-top: 0.65rem;
    font-weight: 600;
    flex-wrap: wrap;
  }

  .tier-label {
    margin-left: auto;
    color: var(--accent);
    font-weight: 700;
  }

  .eco-badge {
    font-size: 0.7rem;
    font-weight: 800;
    text-transform: uppercase;
    padding: 0.22rem 0.55rem;
    border-radius: 999px;
    letter-spacing: 0.04em;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-secondary);
  }

  .eco-badge.npm {
    border-color: rgba(203, 56, 55, 0.35);
    color: #b91c1c;
    background: rgba(203, 56, 55, 0.06);
  }

  .eco-badge.go {
    border-color: rgba(0, 173, 216, 0.35);
    color: #0369a1;
    background: rgba(0, 173, 216, 0.06);
  }

  .eco-badge.cargo {
    border-color: rgba(222, 104, 40, 0.35);
    color: #c2410c;
    background: rgba(222, 104, 40, 0.06);
  }

  .eco-badge.pypi {
    border-color: rgba(55, 148, 225, 0.35);
    color: #1d4ed8;
    background: rgba(55, 148, 225, 0.06);
  }

  .detail-card {
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    border-radius: 14px;
    padding: 1.5rem;
  }

  .back-button {
    background: none;
    border: none;
    color: var(--accent);
    cursor: pointer;
    font-size: 0.88rem;
    font-weight: 700;
    padding: 0;
    margin-bottom: 1.1rem;
  }

  .back-button:hover {
    text-decoration: underline;
  }

  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1.25rem;
    gap: 1rem;
  }

  .detail-header h2 {
    font-size: 1.5rem;
    font-weight: 900;
    margin: 0 0 0.4rem 0;
    color: var(--text-primary);
  }

  .trust-circle {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 68px;
    height: 68px;
    border-radius: 50%;
    border: 3px solid var(--tc);
    flex-shrink: 0;
    background: transparent;
  }

  .trust-num {
    font-weight: 900;
    font-size: 1.2rem;
    color: var(--tc);
    line-height: 1;
  }

  .trust-lbl {
    font-size: 0.65rem;
    font-weight: 700;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .detail-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.85rem;
  }

  .detail-item {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 0.8rem;
  }

  .detail-item.full-width {
    grid-column: span 2;
  }

  .label {
    font-size: 0.7rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-secondary);
    margin-bottom: 0.3rem;
  }

  .value {
    color: var(--text-primary);
    font-weight: 600;
    font-size: 0.95rem;
  }

  .mono {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
      monospace;
    font-size: 0.85rem;
  }

  .badge-latest {
    background: #16a34a;
    color: white;
    padding: 0.18rem 0.45rem;
    border-radius: 6px;
    font-size: 0.72rem;
    font-weight: 800;
    margin-left: 0.4rem;
    vertical-align: middle;
  }

  .notes {
    color: var(--text-primary);
    line-height: 1.6;
    font-size: 0.9rem;
    margin: 0;
  }

  .tags-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .tag-item {
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.35rem 0.65rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .tag-label {
    font-weight: 800;
    color: var(--accent);
    font-size: 0.85rem;
  }

  .tag-value {
    font-weight: 600;
    color: var(--text-secondary);
    font-size: 0.85rem;
  }

  footer {
    text-align: center;
    color: var(--text-secondary);
    font-size: 0.78rem;
    padding: 1.5rem 0;
    font-weight: 600;
  }

  @media (max-width: 700px) {
    .search-row {
      grid-template-columns: 1fr;
    }

    .btn-search,
    .btn-add {
      width: 100%;
    }

    .detail-grid {
      grid-template-columns: 1fr;
    }

    .detail-item.full-width {
      grid-column: span 1;
    }

    .tier-label {
      margin-left: 0;
    }

    .topbar-right {
      flex-wrap: wrap;
      justify-content: flex-end;
    }
  }
</style>
