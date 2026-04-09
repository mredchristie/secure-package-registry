<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { searchAPI } from "$lib/api";
  import type {
    PackageVersionDetail,
    VerifyResponse,
    VersionSummary,
  } from "$lib/types/api";

  // Extract route params — [...identifier] is a rest param for scoped packages
  const ecosystem = $derived(page.params.ecosystem ?? "");
  const identifier = $derived(decodeURIComponent(page.params.identifier ?? ""));
  const version = $derived(page.params.version ?? "");

  let pkg = $state<PackageVersionDetail | null>(null);
  let versions = $state<VersionSummary[]>([]);
  let verifyResult = $state<VerifyResponse | null>(null);
  let loading = $state(true);
  let verifying = $state(false);
  let error = $state("");

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

  // Count how many checks pass (0-3): attestation, oss rebuild, behavior
  function checksPassedCount(
    v: VersionSummary | null,
    verify: VerifyResponse | null,
  ): number {
    if (!v && !verify) return 0;
    let count = 0;
    if (verify) {
      if (verify.upstream_attestation) count++;
      if (verify.oss_rebuild) count++;
    } else if (v) {
      if (v.has_attestation) count++;
      if (v.has_oss_rebuild) count++;
    }
    // behavior from version list
    if (v?.behavior_passed) count++;
    return count;
  }

  function checksColor(count: number): string {
    if (count === 3) return "checks-all";
    if (count >= 1) return "checks-some";
    return "checks-none";
  }

  // Find the current version's summary from the versions list
  const currentVersionSummary = $derived(
    versions.find((v) => v.version === version) ?? null,
  );

  const passedCount = $derived(
    checksPassedCount(currentVersionSummary, verifyResult),
  );

  async function loadData() {
    loading = true;
    error = "";
    verifyResult = null;

    try {
      // If version is "latest", resolve to the actual latest version number
      if (version === "latest") {
        const versionList = await searchAPI.listVersions(ecosystem, identifier);
        const latest = versionList.versions.find((v) => v.latest);
        if (latest) {
          goto(
            `/packages/${ecosystem}/${encodeURIComponent(identifier)}/${latest.version}`,
            { replaceState: true },
          );
          return;
        }
      }

      const [versionDetail, versionList] = await Promise.all([
        searchAPI.getVersion(ecosystem, identifier, version),
        searchAPI.listVersions(ecosystem, identifier),
      ]);
      pkg = versionDetail;
      versions = versionList.versions;
    } catch {
      error = "Failed to load package details.";
      pkg = null;
    } finally {
      loading = false;
    }
  }

  async function runVerification() {
    verifying = true;
    try {
      verifyResult = await searchAPI.verify(ecosystem, identifier, version);
    } catch {
      // Silently fail — verification is best-effort
    } finally {
      verifying = false;
    }
  }

  $effect(() => {
    if (ecosystem && identifier && version) {
      loadData();
    }
  });
</script>

<main>
  <a href="/search" class="back-link">Back to search</a>

  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  {#if loading}
    <div class="state-msg">Loading…</div>
  {:else if pkg}
    <!-- Header -->
    <div class="pkg-header">
      <div class="pkg-title-row">
        <h1 class="pkg-name">{pkg.identifier}</h1>
        <span class="eco-badge {pkg.ecosystem}">{pkg.ecosystem}</span>
      </div>
      <div class="pkg-meta">
        <span class="version-pill">
          {pkg.version}
          {#if pkg.latest}
            <span class="latest-badge">latest</span>
          {/if}
        </span>
        <span class="checks-pill {checksColor(passedCount)}">
          {passedCount}/3 checks
        </span>
      </div>
    </div>

    <div class="layout">
      <!-- Main content -->
      <div class="main-col">
        <!-- Verification -->
        <section class="card">
          <div class="card-header-row">
            <h2 class="card-title">Verification</h2>
            <button
              onclick={runVerification}
              disabled={verifying}
              class="verify-btn"
            >
              {verifying ? "Verifying…" : "Run verification"}
            </button>
          </div>

          <div class="check-grid">
            {#each [{ label: "Upstream attestation", passed: verifyResult?.upstream_attestation ?? currentVersionSummary?.has_attestation ?? false }, { label: "OSS reproducible build", passed: verifyResult?.oss_rebuild ?? currentVersionSummary?.has_oss_rebuild ?? false }, { label: "Behavioral analysis", passed: currentVersionSummary?.behavior_passed ?? false }] as check}
              <div class="check-row">
                <span class="check-indicator {check.passed ? 'pass' : 'fail'}">
                  {check.passed ? "Pass" : "Fail"}
                </span>
                <span class="check-label">{check.label}</span>
              </div>
            {/each}
          </div>
        </section>

        <!-- Tags -->
        {#if pkg.tags.length > 0}
          <section class="card">
            <h2 class="card-title">Tags</h2>
            <div class="tag-list">
              {#each pkg.tags as tag}
                <div class="tag-row">
                  <span class="tag-label">{tag.label}</span>
                  <span class="tag-value">{decodeTagValue(tag.data)}</span>
                </div>
              {/each}
            </div>
          </section>
        {/if}

        <!-- Source -->
        <section class="card">
          <h2 class="card-title">Source</h2>
          <div class="source-list">
            {#if pkg.source.url}
              <div class="source-row">
                <span class="source-label">Repository</span>
                <a href={pkg.source.url} class="source-link" target="_blank">
                  {pkg.source.url}
                </a>
              </div>
            {/if}
            {#if pkg.source.tag}
              <div class="source-row">
                <span class="source-label">Tag</span>
                <code class="source-code">{pkg.source.tag}</code>
              </div>
            {/if}
            {#if pkg.source.commit}
              <div class="source-row">
                <span class="source-label">Commit</span>
                <code class="source-code">{pkg.source.commit}</code>
              </div>
            {/if}
            <div class="source-row">
              <span class="source-label">Install</span>
              <code class="source-code">
                npm install {pkg.identifier}@{pkg.version}
              </code>
            </div>
          </div>
        </section>
      </div>

      <!-- Sidebar: Versions -->
      <aside class="side-col">
        <section class="card">
          <h2 class="card-title">
            Versions
            <span class="version-count">{versions.length}</span>
          </h2>
          <div class="version-list">
            {#each versions as v}
              <a
                href="/packages/{ecosystem}/{encodeURIComponent(
                  identifier,
                )}/{v.version}"
                class="version-row"
                class:active={v.version === version}
              >
                <span class="version-name">
                  {v.version}
                  {#if v.latest}
                    <span class="latest-badge sm">latest</span>
                  {/if}
                </span>
                <span
                  class="checks-pill sm {checksColor(
                    (v.has_attestation ? 1 : 0) +
                      (v.has_oss_rebuild ? 1 : 0) +
                      (v.behavior_passed ? 1 : 0),
                  )}"
                >
                  {(v.has_attestation ? 1 : 0) +
                    (v.has_oss_rebuild ? 1 : 0) +
                    (v.behavior_passed ? 1 : 0)}/3
                </span>
              </a>
            {:else}
              <p class="empty-text">No versions found.</p>
            {/each}
          </div>
        </section>
      </aside>
    </div>
  {/if}
</main>

<style>
  main {
    max-width: 960px;
    margin: 0 auto;
    padding: 1.5rem 1.25rem 4rem;
    color: var(--text-primary);
  }

  .back-link {
    display: inline-block;
    margin-bottom: 1rem;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-secondary);
    text-decoration: none;
    transition: color 0.15s;
  }

  .back-link:hover {
    color: var(--accent);
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

  .state-msg {
    text-align: center;
    color: var(--text-secondary);
    padding: 3rem 0;
    font-size: 0.9rem;
  }

  /* Header */
  .pkg-header {
    margin-bottom: 1.25rem;
  }

  .pkg-title-row {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    margin-bottom: 0.5rem;
  }

  .pkg-name {
    font-size: 1.5rem;
    font-weight: 800;
    margin: 0;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .pkg-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .version-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.2rem 0.6rem;
    font-size: 0.8rem;
    font-weight: 700;
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    border-radius: 6px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text-secondary);
  }

  .latest-badge {
    display: inline-block;
    padding: 0.1rem 0.35rem;
    font-size: 0.6rem;
    font-weight: 800;
    border-radius: 4px;
    background: #16a34a;
    color: white;
    text-transform: uppercase;
    font-family:
      system-ui,
      -apple-system,
      sans-serif;
  }

  .latest-badge.sm {
    font-size: 0.55rem;
    padding: 0.05rem 0.25rem;
  }

  /* Checks pill */
  .checks-pill {
    display: inline-block;
    padding: 0.2rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 800;
    border-radius: 6px;
    border: 1.5px solid;
  }

  .checks-pill.sm {
    font-size: 0.65rem;
    padding: 0.1rem 0.35rem;
  }

  .checks-all {
    color: #16a34a;
    border-color: #22c55e;
    background: rgba(22, 163, 74, 0.08);
  }

  .checks-some {
    color: #f59e0b;
    border-color: #fbbf24;
    background: rgba(245, 158, 11, 0.08);
  }

  .checks-none {
    color: #dc2626;
    border-color: #ef4444;
    background: rgba(220, 38, 38, 0.08);
  }

  /* Ecosystem badge */
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

  /* Layout */
  .layout {
    display: grid;
    grid-template-columns: 1fr 280px;
    gap: 1rem;
    align-items: start;
  }

  .main-col,
  .side-col {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  /* Cards */
  .card {
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    border-radius: 10px;
    padding: 1rem;
  }

  .card-header-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.75rem;
  }

  .card-title {
    font-size: 0.9rem;
    font-weight: 800;
    margin: 0 0 0.75rem 0;
    color: var(--text-primary);
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .card-header-row .card-title {
    margin-bottom: 0;
  }

  .version-count {
    font-size: 0.7rem;
    font-weight: 700;
    padding: 0.1rem 0.35rem;
    border-radius: 4px;
    background: var(--bg-secondary);
    color: var(--text-secondary);
  }

  /* Verification */
  .verify-btn {
    padding: 0.35rem 0.7rem;
    font-size: 0.75rem;
    font-weight: 700;
    border-radius: 6px;
    border: 1px solid var(--accent);
    background: var(--accent);
    color: white;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .verify-btn:hover:not(:disabled) {
    opacity: 0.85;
  }

  .verify-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .check-grid {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .check-row {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }

  .check-indicator {
    display: inline-block;
    width: 3rem;
    text-align: center;
    padding: 0.15rem 0;
    font-size: 0.7rem;
    font-weight: 800;
    border-radius: 4px;
    text-transform: uppercase;
    flex-shrink: 0;
  }

  .check-indicator.pass {
    background: rgba(22, 163, 74, 0.12);
    color: #16a34a;
  }

  .check-indicator.fail {
    background: rgba(220, 38, 38, 0.1);
    color: #dc2626;
  }

  .check-label {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  /* Tags */
  .tag-list {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .tag-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.35rem 0;
    border-bottom: 1px solid var(--border);
  }

  .tag-row:last-child {
    border-bottom: none;
  }

  .tag-label {
    font-size: 0.8rem;
    font-weight: 700;
    color: var(--text-secondary);
  }

  .tag-value {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-primary);
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  }

  /* Source */
  .source-list {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
  }

  .source-row {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .source-label {
    font-size: 0.65rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
  }

  .source-link {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--accent);
    text-decoration: none;
    word-break: break-all;
  }

  .source-link:hover {
    text-decoration: underline;
  }

  .source-code {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-primary);
    word-break: break-all;
  }

  /* Version list sidebar */
  .version-list {
    display: flex;
    flex-direction: column;
    max-height: 400px;
    overflow-y: auto;
  }

  .version-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.4rem;
    padding: 0.4rem 0.5rem;
    border-radius: 6px;
    text-decoration: none;
    color: inherit;
    transition: background 0.1s;
  }

  .version-row:hover {
    background: var(--bg-secondary);
  }

  .version-row.active {
    background: rgba(29, 78, 216, 0.08);
    border: 1px solid rgba(29, 78, 216, 0.15);
  }

  .version-name {
    font-size: 0.8rem;
    font-weight: 600;
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    color: var(--text-primary);
    display: flex;
    align-items: center;
    gap: 0.3rem;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty-text {
    font-size: 0.8rem;
    color: var(--text-secondary);
    padding: 0.5rem 0;
    margin: 0;
  }

  @media (max-width: 768px) {
    .layout {
      grid-template-columns: 1fr;
    }
  }
</style>
