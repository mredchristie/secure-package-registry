<script lang="ts">
  import { onMount } from 'svelte';
  
  // Types matching the API
  interface PackageSummary {
    identifier: string;
    ecosystem: 'npm' | 'go' | 'cargo' | 'pypi';
    latest_version: string;
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
      value_type: 'boolean' | 'integer' | 'float';
      data: string; // base64 encoded JSON
    }>;
  }
  
  // Decode base64 and parse JSON value
  function decodeTagValue(data: string): string {
    try {
      const decoded = atob(data);
      const parsed = JSON.parse(decoded);
      return String(parsed);
    } catch (e) {
      return data;
    }
  }
  
  let searchQuery = '';
  let selectedEcosystem: string = '';
  let searchResults: PackageSummary[] = [];
  let selectedPackage: PackageVersion | null = null;
  let loading = false;
  let error = '';
  
  const ecosystems = [
    { value: '', label: 'All Ecosystems' },
    { value: 'npm', label: 'npm' },
    { value: 'go', label: 'Go' },
    { value: 'cargo', label: 'Cargo' },
    { value: 'pypi', label: 'PyPI' }
  ];
  
  async function searchPackages() {
    if (!searchQuery.trim()) {
      error = 'Please enter a search term';
      return;
    }
    
    loading = true;
    error = '';
    selectedPackage = null;
    
    try {
      const params = new URLSearchParams();
      params.append('q', searchQuery);
      if (selectedEcosystem) {
        params.append('ecosystem', selectedEcosystem);
      }
      
      const response = await fetch(`/api/v1/svc/packages?${params}`);
      if (!response.ok) {
        throw new Error('Search failed');
      }
      
      const data: SearchResult = await response.json();
      searchResults = data.items;
    } catch (err) {
      error = 'Failed to search packages. Please try again.';
      searchResults = [];
    } finally {
      loading = false;
    }
  }
  
  async function selectPackage(pkg: PackageSummary) {
    loading = true;
    error = '';
    
    try {
      const response = await fetch(`/api/v1/svc/packages/${pkg.ecosystem}/${pkg.identifier}/${pkg.latest_version}`);
      if (!response.ok) {
        throw new Error('Failed to fetch package details');
      }
      
      selectedPackage = await response.json();
    } catch (err) {
      error = 'Failed to fetch package details.';
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
    if (event.key === 'Enter') {
      searchPackages();
    }
  }
</script>

<main>
  <h1>Package Registry Search</h1>
  
  {#if error}
    <div class="error">{error}</div>
  {/if}
  
  {#if selectedPackage}
    <div class="package-details">
      <button class="back-button" on:click={clearSelection}>← Back to search</button>
      
      <h2>{selectedPackage.identifier}</h2>
      <div class="detail-grid">
        <div class="detail-item">
          <span class="label">Ecosystem:</span>
          <span class="value">{selectedPackage.ecosystem}</span>
        </div>
        <div class="detail-item">
          <span class="label">Version:</span>
          <span class="value">{selectedPackage.version}</span>
          {#if selectedPackage.latest}
            <span class="badge latest">Latest</span>
          {/if}
        </div>
        <div class="detail-item">
          <span class="label">Trust Level:</span>
          <span class="value">{selectedPackage.trust_level}</span>
        </div>
        <div class="detail-item full-width">
          <span class="label">Source URL:</span>
          <a href={selectedPackage.source.url} target="_blank" rel="noopener">{selectedPackage.source.url}</a>
        </div>
        {#if selectedPackage.source.tag}
          <div class="detail-item">
            <span class="label">Tag:</span>
            <span class="value">{selectedPackage.source.tag}</span>
          </div>
        {/if}
        {#if selectedPackage.source.commit}
          <div class="detail-item">
            <span class="label">Commit:</span>
            <span class="value mono">{selectedPackage.source.commit}</span>
          </div>
        {/if}
        {#if selectedPackage.maintainer_notes}
          <div class="detail-item full-width">
            <span class="label">Maintainer Notes:</span>
            <p class="notes">{selectedPackage.maintainer_notes}</p>
          </div>
        {/if}
        {#if selectedPackage.tags.length > 0}
          <div class="detail-item full-width">
            <span class="label">Tags:</span>
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
    <div class="search-container">
      <div class="search-row">
        <input
          type="text"
          bind:value={searchQuery}
          on:keydown={handleKeydown}
          placeholder="Search packages..."
          class="search-input"
        />
        <select bind:value={selectedEcosystem} class="ecosystem-select">
          {#each ecosystems as eco}
            <option value={eco.value}>{eco.label}</option>
          {/each}
        </select>
        <button on:click={searchPackages} disabled={loading} class="search-button">
          {loading ? 'Searching...' : 'Search'}
        </button>
      </div>
      
      {#if loading}
        <div class="loading">Searching...</div>
      {:else if searchResults.length > 0}
        <div class="results">
          <h3>Search Results</h3>
          <ul class="results-list">
            {#each searchResults as pkg}
              <li>
                <button class="result-item" on:click={() => selectPackage(pkg)}>
                  <div class="result-header">
                    <span class="identifier">{pkg.identifier}</span>
                    <span class="ecosystem-badge">{pkg.ecosystem}</span>
                  </div>
                  <div class="result-version">Latest: {pkg.latest_version}</div>
                </button>
              </li>
            {/each}
          </ul>
        </div>
      {:else if searchQuery && !loading}
        <div class="no-results">No packages found. Try a different search term.</div>
      {/if}
    </div>
  {/if}
</main>

<style>
  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
    margin: 0;
    padding: 0;
    background: #f5f5f5;
  }
  
  main {
    max-width: 800px;
    margin: 0 auto;
    padding: 2rem;
  }
  
  h1 {
    color: #333;
    margin-bottom: 2rem;
  }
  
  .error {
    background: #ffebee;
    color: #c62828;
    padding: 1rem;
    border-radius: 4px;
    margin-bottom: 1rem;
  }
  
  .search-container {
    background: white;
    padding: 2rem;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  }
  
  .search-row {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
  }
  
  .search-input {
    flex: 1;
    padding: 0.75rem 1rem;
    font-size: 1rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    outline: none;
  }
  
  .search-input:focus {
    border-color: #1976d2;
  }
  
  .ecosystem-select {
    padding: 0.75rem;
    font-size: 1rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    background: white;
    cursor: pointer;
  }
  
  .search-button {
    padding: 0.75rem 1.5rem;
    font-size: 1rem;
    background: #1976d2;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.2s;
  }
  
  .search-button:hover:not(:disabled) {
    background: #1565c0;
  }
  
  .search-button:disabled {
    background: #ccc;
    cursor: not-allowed;
  }
  
  .loading {
    text-align: center;
    color: #666;
    padding: 2rem;
  }
  
  .results h3 {
    margin-top: 0;
    color: #333;
    border-bottom: 1px solid #eee;
    padding-bottom: 0.5rem;
  }
  
  .results-list {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  
  .results-list li {
    margin-bottom: 0.5rem;
  }
  
  .result-item {
    width: 100%;
    padding: 1rem;
    background: #f8f9fa;
    border: 1px solid #e0e0e0;
    border-radius: 4px;
    text-align: left;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .result-item:hover {
    background: #e3f2fd;
    border-color: #1976d2;
  }
  
  .result-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.25rem;
  }
  
  .identifier {
    font-weight: 600;
    font-size: 1.1rem;
    color: #333;
  }
  
  .ecosystem-badge {
    background: #e0e0e0;
    color: #616161;
    padding: 0.25rem 0.5rem;
    border-radius: 12px;
    font-size: 0.75rem;
    text-transform: uppercase;
    font-weight: 500;
  }
  
  .result-version {
    color: #666;
    font-size: 0.9rem;
  }
  
  .no-results {
    text-align: center;
    color: #666;
    padding: 2rem;
  }
  
  .package-details {
    background: white;
    padding: 2rem;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  }
  
  .back-button {
    background: none;
    border: none;
    color: #1976d2;
    cursor: pointer;
    font-size: 1rem;
    padding: 0;
    margin-bottom: 1rem;
  }
  
  .back-button:hover {
    text-decoration: underline;
  }
  
  h2 {
    margin-top: 0;
    color: #333;
    border-bottom: 2px solid #1976d2;
    padding-bottom: 0.5rem;
  }
  
  .detail-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
    margin-top: 1.5rem;
  }
  
  .detail-item {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  
  .detail-item.full-width {
    grid-column: span 2;
  }
  
  .label {
    font-weight: 600;
    color: #666;
    font-size: 0.875rem;
    text-transform: uppercase;
    letter-spacing: 0.025em;
  }
  
  .value {
    color: #333;
    font-size: 1rem;
  }
  
  .mono {
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    font-size: 0.875rem;
  }
  
  .badge {
    display: inline-block;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
    margin-left: 0.5rem;
  }
  
  .badge.latest {
    background: #4caf50;
    color: white;
  }
  
  a {
    color: #1976d2;
    text-decoration: none;
    word-break: break-all;
  }
  
  a:hover {
    text-decoration: underline;
  }
  
  .notes {
    margin: 0;
    color: #333;
    line-height: 1.5;
    background: #f5f5f5;
    padding: 0.75rem;
    border-radius: 4px;
  }
  
  .tags-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  
  .tag-item {
    background: #e3f2fd;
    border: 1px solid #bbdefb;
    padding: 0.5rem 0.75rem;
    border-radius: 4px;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  
  .tag-label {
    font-weight: 600;
    color: #1976d2;
  }
  
  .tag-value {
    font-weight: 500;
    color: #333;
  }
  
</style>
