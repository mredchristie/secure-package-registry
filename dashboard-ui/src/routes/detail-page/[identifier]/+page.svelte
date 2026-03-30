<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/state";

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
      data: string;
    }>;
  }

  type DetailTab = "Read Me" | "Dependents" | "Dependencies" | "Versions";

  let selectedPackage: PackageVersion | null = null;
  let loading = true;
  let error = "";
  let activeTab: DetailTab = "Read Me";
  let publishedAt = "Unknown";

  function trustColor(score: number): string {
    if (score >= 90) return "#16a34a";
    if (score >= 70) return "#f59e0b";
    return "#dc2626";
  }

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

  function setActiveTab(tab: DetailTab) {
    activeTab = tab;
  }

  function getTabContent(tab: DetailTab): string[] {
    if (!selectedPackage) return [];

    switch (tab) {
      case "Read Me":
        return [
          selectedPackage.maintainer_notes ||
            "No readme information available.",
        ];

      case "Dependents":
        return ["Dependents data not available yet."];

      case "Dependencies":
        return selectedPackage.tags.length > 0
          ? selectedPackage.tags.map(
              (tag) => `• ${tag.label}: ${decodeTagValue(tag.data)}`,
            )
          : ["No dependency data available."];

      case "Versions":
        return ["Version history not available yet."];

      default:
        return [];
    }
  }

  async function loadPackageDetails() {
    loading = true;
    error = "";

    try {
      const rawIdentifier = page.params.identifier;

      if (!rawIdentifier) {
        throw new Error("Missing package identifier in URL.");
      }

      const identifier = decodeURIComponent(rawIdentifier);
      const ecosystem = page.url.searchParams.get("ecosystem");
      const version = page.url.searchParams.get("version");

      if (!ecosystem || !version) {
        throw new Error("Missing ecosystem or version in URL.");
      }

      const safeIdentifier = encodeURIComponent(identifier);

      const response = await fetch(
        `/api/v1/svc/packages/${ecosystem}/${safeIdentifier}/${version}`,
      );

      if (!response.ok) {
        throw new Error("Failed to fetch package details.");
      }

      selectedPackage = await response.json();
      activeTab = "Read Me";
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Failed to load package details.";
      selectedPackage = null;
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    await loadPackageDetails();
  });
</script>

<main class="main">
  {#if error}
    <div class="error">{error}</div>
  {/if}

  {#if loading}
    <div class="state-msg">Loading package details…</div>
  {:else if selectedPackage}
    <div class="detail-page">
      <div class="detail-hero">
        <div class="hero-left">
          <h1 class="package-title">{selectedPackage.identifier}</h1>

          <div class="package-meta-row">
            <span class="eco-badge {selectedPackage.ecosystem}">
              {selectedPackage.ecosystem}
            </span>

            <span class="package-meta-pill">
              Version {selectedPackage.version}
              {#if selectedPackage.latest}
                <span class="badge-latest">Latest</span>
              {/if}
            </span>

            <span class="package-meta-pill">
              Published {publishedAt}
            </span>
          </div>
        </div>

        <div
          class="trust-circle"
          style="--tc: {trustColor(selectedPackage.trust_level)}"
        >
          <span class="trust-num">{selectedPackage.trust_level}</span>
          <span class="trust-lbl">Trust</span>
        </div>
      </div>

      <div class="detail-layout">
        <div class="detail-main">
          <div class="tabs-bar">
            <button
              class="tab-button"
              class:active-tab={activeTab === "Read Me"}
              type="button"
              onclick={() => setActiveTab("Read Me")}
            >
              Readme
            </button>

            <button
              class="tab-button"
              class:active-tab={activeTab === "Dependents"}
              type="button"
              onclick={() => setActiveTab("Dependents")}
            >
              Dependents
            </button>

            <button
              class="tab-button"
              class:active-tab={activeTab === "Dependencies"}
              type="button"
              onclick={() => setActiveTab("Dependencies")}
            >
              Dependencies
            </button>

            <button
              class="tab-button"
              class:active-tab={activeTab === "Versions"}
              type="button"
              onclick={() => setActiveTab("Versions")}
            >
              Versions
            </button>
          </div>

          <section class="main-card tab-content-card">
            <h3 class="section-title">{activeTab}</h3>

            <div class="tab-content">
              {#each getTabContent(activeTab) as line}
                <p>{line}</p>
              {/each}
            </div>
          </section>
        </div>

        <aside class="detail-sidebar">
          <section class="side-card risk-card">
            <h3 class="section-title">Risk Profile</h3>
            <div class="risk-content">
              <p>
                <strong>Verification ({selectedPackage.source.tag})</strong>
              </p>
              <p>Reproducible</p>
              <p>Behavior clean</p>
              <p>Last verified recently</p>
            </div>
          </section>

          <section class="side-card action-card">
            <button class="upgrade-button" type="button">
              Request check<br />
              Tier upgrade
            </button>
          </section>

          <section class="side-card links-card">
            <h3 class="section-title">Links</h3>
            <div class="link-list">
              <div class="link-row">
                <span class="label">Install</span>
                <span class="value mono">
                  npm i {selectedPackage.identifier}
                </span>
              </div>

              <div class="link-row">
                <span class="label">Repository</span>
                <span class="value mono">{selectedPackage.source.url}</span>
              </div>

              <div class="link-row">
                <span class="label">Tag</span>
                <span class="value mono">{selectedPackage.source.tag}</span>
              </div>

              <!-- <div class="link-row">
                <span class="label">Commit</span>
                <span class="value mono">{selectedPackage.source.commit}</span>
              </div> -->
            </div>
          </section>

          <!-- <section class="side-card file-card">
            <h3 class="section-title">File changes</h3>
            <div class="file-change-body">
              <p>No recent breaking changes detected.</p>
            </div>
          </section> -->
        </aside>
      </div>
    </div>
  {/if}
</main>

<style>
  .main {
    max-width: 1000px;
    margin: 0 auto;
    padding: 1.5rem 1.25rem 4rem;
    color: var(--text-primary);
  }

  .error {
    background: rgba(220, 38, 38, 0.08);
    border: 1px solid rgba(220, 38, 38, 0.2);
    color: #b91c1c;
    padding: 0.9rem 1rem;
    border-radius: 12px;
    margin-bottom: 1rem;
  }

  .state-msg {
    text-align: center;
    color: var(--text-secondary);
    padding: 3rem 0;
    font-size: 0.95rem;
  }

  .detail-page {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .hero-left {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .detail-hero {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    border-radius: 14px;
    padding: 1.25rem;
  }

  .package-title {
    font-size: 2rem;
    font-weight: 900;
    color: var(--text-primary);
    margin: 0;
  }

  .package-meta-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.65rem;
    align-items: center;
  }

  .package-meta-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.45rem 0.75rem;
    border-radius: 999px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    font-size: 0.88rem;
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

  .badge-latest {
    background: #16a34a;
    color: white;
    padding: 0.18rem 0.45rem;
    border-radius: 6px;
    font-size: 0.72rem;
    font-weight: 800;
    margin-left: 0.2rem;
    vertical-align: middle;
  }

  .tabs-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.65rem;
  }

  .tab-button {
    padding: 0.7rem 1rem;
    border-radius: 10px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-secondary);
    font-size: 0.9rem;
    font-weight: 700;
    cursor: pointer;
    transition:
      background 0.15s,
      border-color 0.15s,
      color 0.15s;
  }

  .tab-button:hover {
    border-color: var(--accent);
    color: var(--text-primary);
  }

  .active-tab {
    background: rgba(29, 78, 216, 0.08);
    border-color: var(--accent);
    color: var(--accent);
  }

  .detail-layout {
    display: grid;
    grid-template-columns: 1.6fr 1fr;
    gap: 1rem;
    align-items: start;
  }

  .detail-main,
  .detail-sidebar {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .main-card,
  .side-card {
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    border-radius: 14px;
    padding: 1rem;
  }

  .tab-content-card {
    min-height: 720px;
  }

  .section-title {
    font-size: 1.05rem;
    font-weight: 800;
    color: var(--text-primary);
    margin: 0 0 0.85rem 0;
  }

  .tab-content {
    display: flex;
    flex-direction: column;
    gap: 0.65rem;
    color: var(--text-primary);
    line-height: 1.6;
  }

  .tab-content p {
    margin: 0;
  }

  .risk-content,
  .link-list,
  .file-change-body {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .link-row {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
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

  .upgrade-button {
    width: 100%;
    min-height: 72px;
    border: 1px solid var(--accent);
    border-radius: 12px;
    background: transparent;
    color: var(--accent);
    font-weight: 800;
    font-size: 1rem;
    cursor: pointer;
    line-height: 1.35;
  }

  .upgrade-button:hover {
    background: rgba(29, 78, 216, 0.08);
  }

  @media (max-width: 900px) {
    .detail-layout {
      grid-template-columns: 1fr;
    }

    .tab-content-card {
      min-height: auto;
    }
  }
</style>
