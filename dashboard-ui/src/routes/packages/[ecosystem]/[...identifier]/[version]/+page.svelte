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
  let error = $state("");

  // Trust score (0-3): provenance (grouped), behavioral analysis, manual review
  // Returns null if behavioral analysis hasn't been run yet
  function checksPassedCount(
    v: VersionSummary | null,
    verify: VerifyResponse | null,
  ): number | null {
    if (!v && !verify) return 0;

    // Manual approval always gives full trust
    if (v?.manually_approved === true) {
      return 3;
    }

    // Provenance is a group: any one of attestation, oss rebuild, or reproducible
    const hasProvenance =
      (verify?.upstream_attestation ?? v?.has_attestation ?? false) ||
      (verify?.oss_rebuild ?? v?.has_oss_rebuild ?? false) ||
      (v?.has_reproducible ?? false);
    const behaviorPassed = v?.behavior_passed;

    // If behavioral analysis hasn't been run, score is indeterminate
    if (behaviorPassed === null) {
      return null;
    }

    // Both pass = full trust
    if (behaviorPassed && hasProvenance) return 3;
    // Only behavior passes, missing provenance is common
    if (behaviorPassed && !hasProvenance) return 2;
    // Only provenance passes, failed behavior is a red flag
    if (!behaviorPassed && hasProvenance) return 1;
    // Neither passes
    return 0;
  }

  function checksColor(count: number): string {
    if (count === 3) return "checks-all";
    if (count >= 1) return "checks-some";
    return "checks-none";
  }

  // Calculate trust score for a version (used in version sidebar list)
  // Returns null if behavioral analysis is pending
  function calculateTrustScore(v: VersionSummary): number | null {
    // Manual approval always gives full trust
    if (v.manually_approved === true) {
      return 3;
    }

    const hasProvenance =
      v.has_attestation || v.has_oss_rebuild || v.has_reproducible;
    const behaviorPassed = v.behavior_passed;

    // If behavioral analysis hasn't been run, return null to show "Pending"
    if (behaviorPassed === null) {
      return null;
    }

    if (behaviorPassed && hasProvenance) return 3;
    if (behaviorPassed && !hasProvenance) return 2;
    if (!behaviorPassed && hasProvenance) return 1;
    return 0;
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

  // Load data when component mounts or route params change
  $effect(() => {
    // Re-run when route params change
    ecosystem;
    identifier;
    version;
    loadData();
  });
</script>

<svelte:head>
  <title>{pkg ? `${pkg.identifier}@${version}` : "Package"} - SPR</title>
</svelte:head>

<main>
  <a href="/search" class="back-link">Back to search</a>

  {#if loading && !pkg}
    <div class="loading-container">
      <div class="spinner"></div>
    </div>
  {:else if error && !pkg}
    <div class="error-banner">
      <div class="error-content">
        <span class="error-icon">&#9888;</span>
        <span>{error}</span>
      </div>
    </div>
  {:else if pkg}
    <div class="header">
      <div class="header-left">
        <h1 class="package-title">
          {pkg.identifier}
          <span class="eco-badge">{pkg.ecosystem}</span>
        </h1>
        <span class="version-tag">
          <code class="version-code">{pkg.version}</code>
          {#if pkg.latest}
            <span class="latest-badge">latest</span>
          {/if}
        </span>
        <span class="checks-pill {checksColor(passedCount ?? 0)}">
          {#if passedCount === null}
            Pending
          {:else}
            Trust score: {passedCount}/3
          {/if}
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
          </div>

          <div class="check-grid">
            <!-- Attestation check -->
            <div class="check-row">
              {#if verifyResult?.upstream_attestation ?? currentVersionSummary?.has_attestation ?? false}
                <span class="check-indicator pass">PASS</span>
              {:else}
                <span class="check-indicator fail">Missing</span>
              {/if}
              <span class="check-label">Upstream attestation</span>
            </div>

            <!-- OSS rebuild check -->
            <div class="check-row">
              {#if verifyResult?.oss_rebuild ?? currentVersionSummary?.has_oss_rebuild ?? false}
                <span class="check-indicator pass">PASS</span>
              {:else}
                <span class="check-indicator fail">Missing</span>
              {/if}
              <span class="check-label">OSS reproducible build</span>
            </div>

            <!-- Reproducible build check -->
            <div class="check-row">
              {#if currentVersionSummary?.has_reproducible}
                <span class="check-indicator pass">PASS</span>
              {:else}
                <span class="check-indicator fail">Missing</span>
              {/if}
              <span class="check-label">Verified reproducible build</span>
            </div>

            <!-- Behavioral analysis check -->
            <div class="check-row">
              {#if currentVersionSummary?.behavior_passed === null}
                <span class="check-indicator pending">PENDING</span>
              {:else if currentVersionSummary?.behavior_passed === true}
                <span class="check-indicator pass">PASS</span>
              {:else}
                <span class="check-indicator fail">FAIL</span>
              {/if}
              <span class="check-label">Behavioral analysis</span>
            </div>

            <!-- Manual review status -->
            <div class="check-row">
              {#if currentVersionSummary?.manually_approved === true}
                <span class="check-indicator pass">APPROVED</span>
              {:else if currentVersionSummary?.manually_approved === false}
                <span class="check-indicator fail">REJECTED</span>
              {:else}
                <span class="check-indicator pending">PENDING</span>
              {/if}
              <span class="check-label">Manual review</span>
            </div>

            <!-- Review comment if exists -->
            {#if currentVersionSummary?.review_comment}
              <div class="review-comment-row">
                <span class="review-comment-label">Review comment:</span>
                <span class="review-comment-text"
                  >{currentVersionSummary.review_comment}</span
                >
              </div>
            {/if}
          </div>
        </section>

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
          </div>

          <div class="install-section">
            <div class="install-label">INSTALL</div>
            <code class="install-code">
              npm install {pkg.identifier}@{pkg.version}
            </code>
          </div>
        </section>
      </div>

      <!-- Sidebar -->
      <aside class="sidebar">
        <!-- Versions -->
        <section class="card">
          <h2 class="card-title">
            Versions
            <span class="version-count">{versions.length}</span>
          </h2>
          <div class="version-list">
            {#each versions as v}
              {@const score = calculateTrustScore(v)}
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
                {#if score === null}
                  <span class="checks-pill sm checks-none">Pending</span>
                {:else}
                  <span class="checks-pill sm {checksColor(score)}">
                    {score}/3
                  </span>
                {/if}
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
  }

  .back-link:hover {
    color: var(--text-primary);
  }

  .loading-container {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 50vh;
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 2px solid var(--border);
    border-top-color: var(--text-primary);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .error-banner {
    background: #fee;
    border: 1px solid #fcc;
    padding: 1rem;
    border-radius: 8px;
    margin-bottom: 1rem;
  }

  .error-content {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .error-icon {
    color: #c33;
    font-size: 1.2rem;
  }

  .header {
    margin-bottom: 1.5rem;
  }

  .header-left {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
  }

  .package-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 1.5rem;
    font-weight: 700;
    margin: 0;
  }

  .eco-badge {
    background: var(--surface);
    color: var(--text-secondary);
    font-size: 0.65rem;
    font-weight: 700;
    padding: 0.25rem 0.4rem;
    border-radius: 4px;
    text-transform: uppercase;
  }

  .version-tag {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    background: var(--surface);
    padding: 0.25rem 0.5rem;
    border-radius: 6px;
    font-size: 0.9rem;
  }

  .version-code {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  .latest-badge {
    background: #16a34a;
    color: white;
    font-size: 0.6rem;
    font-weight: 700;
    padding: 0.15rem 0.35rem;
    border-radius: 4px;
    text-transform: uppercase;
  }

  .checks-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 0.75rem;
    border-radius: 20px;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .checks-pill.checks-none {
    background: #fee;
    color: #991b1b;
  }

  .checks-pill.checks-some {
    background: #fef3c7;
    color: #92400e;
  }

  .checks-pill.checks-all {
    background: #dcfce7;
    color: #166534;
  }

  .layout {
    display: grid;
    grid-template-columns: 1fr 280px;
    gap: 1.25rem;
  }

  @media (max-width: 800px) {
    .layout {
      grid-template-columns: 1fr;
    }
    .sidebar {
      order: -1;
    }
  }

  .card {
    background: var(--card-bg);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1.25rem;
  }

  .card + .card {
    margin-top: 1rem;
  }

  .card-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 1.1rem;
    font-weight: 600;
    margin: 0 0 1rem;
  }

  .card-header-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .card-header-row .card-title {
    margin: 0;
  }

  .verify-btn {
    background: var(--btn-primary-bg);
    color: var(--btn-primary-fg);
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s ease;
  }

  .verify-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .verify-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .check-grid {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .check-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .check-indicator {
    min-width: 90px;
    padding: 0.35rem 0.75rem;
    border-radius: 4px;
    font-size: 0.7rem;
    font-weight: 700;
    text-align: center;
    flex-shrink: 0;
  }

  .check-indicator.pass {
    background: #dcfce7;
    color: #166534;
  }

  .check-indicator.fail {
    background: #fee2e2;
    color: #991b1b;
  }

  .check-indicator.pending {
    background: #fef3c7;
    color: #92400e;
  }

  .check-label {
    font-size: 0.9rem;
    color: var(--text-primary);
  }

  .review-comment-row {
    display: flex;
    align-items: flex-start;
    gap: 0.5rem;
    margin-top: 0.5rem;
    padding-top: 0.75rem;
    border-top: 1px solid var(--border);
  }

  .review-comment-label {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .review-comment-text {
    font-size: 0.85rem;
    color: var(--text-primary);
    font-style: italic;
  }

  .source-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }

  .source-row {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .source-label {
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .source-link {
    color: var(--link-color);
    text-decoration: none;
    font-size: 0.85rem;
  }

  .source-link:hover {
    text-decoration: underline;
  }

  .source-code {
    font-size: 0.8rem;
    color: var(--text-secondary);
    background: var(--surface);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
  }

  .install-section {
    background: var(--surface);
    padding: 0.75rem;
    border-radius: 8px;
    margin-top: 1rem;
  }

  .install-label {
    font-size: 0.65rem;
    font-weight: 700;
    color: var(--text-secondary);
    margin-bottom: 0.35rem;
  }

  .install-code {
    font-size: 0.85rem;
    color: var(--text-primary);
    display: block;
  }

  .version-list {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .version-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0.65rem;
    border-radius: 6px;
    text-decoration: none;
    color: inherit;
  }

  .version-row:hover {
    background: var(--surface);
  }

  .version-row.active {
    background: #eef2ff;
  }

  .version-name {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.85rem;
    font-family: ui-monospace, monospace;
  }

  .latest-badge.sm {
    font-size: 0.55rem;
    padding: 0.1rem 0.25rem;
  }

  .checks-pill.sm {
    padding: 0.2rem 0.5rem;
    font-size: 0.7rem;
  }

  .version-count {
    background: var(--surface);
    color: var(--text-secondary);
    font-size: 0.7rem;
    padding: 0.15rem 0.4rem;
    border-radius: 12px;
  }

  .empty-text {
    font-size: 0.85rem;
    color: var(--text-secondary);
    font-style: italic;
    text-align: center;
    padding: 1rem;
  }
</style>
