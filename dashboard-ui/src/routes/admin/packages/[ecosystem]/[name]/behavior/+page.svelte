<script lang="ts">
  import { page } from "$app/stores";
  import { packagesAPI } from "$lib/api";
  import type {
    ProcessTree,
    ProcessNode,
    ProcessBehaviors,
    ReviewStatusResponse,
  } from "$lib/types/api";
  import {
    ArrowLeft,
    Loader2,
    ShieldCheck,
    ShieldAlert,
    ChevronRight,
    ChevronDown,
    Terminal,
    FileText,
    Globe,
    Wifi,
    LoaderCircle,
    CircleAlert,
    CheckCircle2,
    XCircle,
    Clock,
    MessageSquare,
  } from "lucide-svelte";

  let ecosystem = $derived($page.params.ecosystem ?? "");
  let name = $derived(decodeURIComponent($page.params.name ?? ""));
  let version = $derived($page.url.searchParams.get("version") ?? "");

  type ViewMode = "deduped" | "raw";
  let activeView = $state<ViewMode>("deduped");

  let dedupedTree = $state<ProcessTree | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);

  let rawTree = $state<ProcessTree | null>(null);
  let rawLoading = $state(false);
  let rawError = $state<string | null>(null);

  let tree = $derived(activeView === "deduped" ? dedupedTree : rawTree);

  async function loadBehavior() {
    if (!version) {
      error = "No version specified";
      return;
    }
    loading = true;
    error = null;
    try {
      dedupedTree = await packagesAPI.behavior(ecosystem, name, version);
      expandEverything(dedupedTree);
    } catch (e) {
      if (e instanceof Error) {
        error = e.message;
      } else {
        error = "Failed to load behavioral analysis";
      }
    } finally {
      loading = false;
    }
  }

  async function loadRawBehavior() {
    if (rawTree) return;
    rawLoading = true;
    rawError = null;
    try {
      rawTree = await packagesAPI.behaviorRaw(ecosystem, name, version);
      expandEverything(rawTree);
    } catch (e) {
      if (e instanceof Error) {
        rawError = e.message;
      } else {
        rawError = "Failed to load raw behavioral analysis";
      }
    } finally {
      rawLoading = false;
    }
  }

  function switchView(mode: ViewMode) {
    activeView = mode;
    if (mode === "raw" && !rawTree && !rawLoading) {
      loadRawBehavior();
    }
  }

  $effect(() => {
    if (ecosystem && name && version) {
      loadBehavior();
      loadReviewStatus();
    }
  });

  function countBehaviors(node: ProcessNode): {
    execs: number;
    files: number;
    connections: number;
    dns: number;
  } {
    let execs = node.behaviors.execs?.length ?? 0;
    let files = node.behaviors.files_opened?.length ?? 0;
    let connections = node.behaviors.connections?.length ?? 0;
    let dns = node.behaviors.dns_queries?.length ?? 0;

    for (const child of node.children ?? []) {
      const childCounts = countBehaviors(child);
      execs += childCounts.execs;
      files += childCounts.files;
      connections += childCounts.connections;
      dns += childCounts.dns;
    }

    return { execs, files, connections, dns };
  }

  function hasBehaviors(node: ProcessNode): boolean {
    const b = node.behaviors;
    if (
      (b.execs?.length ?? 0) > 0 ||
      (b.files_opened?.length ?? 0) > 0 ||
      (b.connections?.length ?? 0) > 0 ||
      (b.dns_queries?.length ?? 0) > 0
    ) {
      return true;
    }
    return (node.children ?? []).some(hasBehaviors);
  }

  function nodeBehaviorCount(b: ProcessBehaviors): number {
    return (
      (b.execs?.length ?? 0) +
      (b.files_opened?.length ?? 0) +
      (b.connections?.length ?? 0) +
      (b.dns_queries?.length ?? 0)
    );
  }

  let treeIsEmpty = $derived(tree ? !hasBehaviors(tree.root) : true);

  let totalCounts = $derived(tree ? countBehaviors(tree.root) : null);

  let totalBehaviors = $derived(
    totalCounts
      ? totalCounts.execs +
          totalCounts.files +
          totalCounts.connections +
          totalCounts.dns
      : 0,
  );

  let expandedNodes = $state<Set<string>>(new Set());
  let expandedDetails = $state<Set<string>>(new Set());

  // Review panel state
  let reviewStatus = $state<ReviewStatusResponse | null>(null);
  let reviewLoading = $state(false);
  let reviewError = $state<string | null>(null);
  let reviewApproved = $state(true);
  let reviewComment = $state("");
  let submitting = $state(false);
  let submitError = $state<string | null>(null);
  let submitSuccess = $state<string | null>(null);

  async function loadReviewStatus() {
    if (!version) return;
    reviewLoading = true;
    reviewError = null;
    try {
      reviewStatus = await packagesAPI.getReviewStatus(
        ecosystem,
        name,
        version,
      );
      // Pre-fill form from existing review
      if (reviewStatus.manually_approved !== null) {
        reviewApproved = reviewStatus.manually_approved;
      }
      if (reviewStatus.review_comment) {
        reviewComment = reviewStatus.review_comment;
      }
    } catch (e) {
      reviewError =
        e instanceof Error ? e.message : "Failed to load review status";
    } finally {
      reviewLoading = false;
    }
  }

  async function handleSubmitReview() {
    if (!reviewComment.trim()) {
      submitError = "A review comment is required";
      return;
    }
    submitting = true;
    submitError = null;
    submitSuccess = null;
    try {
      reviewStatus = await packagesAPI.submitReview(ecosystem, name, version, {
        approved: reviewApproved,
        comment: reviewComment.trim(),
      });
      submitSuccess = reviewApproved ? "Version approved" : "Version rejected";
      setTimeout(() => {
        submitSuccess = null;
      }, 3000);
    } catch (e) {
      submitError = e instanceof Error ? e.message : "Failed to submit review";
    } finally {
      submitting = false;
    }
  }

  function toggleNode(identity: string) {
    const next = new Set(expandedNodes);
    if (next.has(identity)) {
      next.delete(identity);
    } else {
      next.add(identity);
    }
    expandedNodes = next;
  }

  function toggleDetails(identity: string) {
    const next = new Set(expandedDetails);
    if (next.has(identity)) {
      next.delete(identity);
    } else {
      next.add(identity);
    }
    expandedDetails = next;
  }

  function expandEverything(t: ProcessTree | null) {
    if (!t) return;
    const nodes = new Set<string>();
    const details = new Set<string>();
    function walk(node: ProcessNode) {
      if ((node.children?.length ?? 0) > 0) {
        nodes.add(node.identity);
      }
      if (nodeBehaviorCount(node.behaviors) > 0) {
        details.add(node.identity);
      }
      for (const child of node.children ?? []) {
        walk(child);
      }
    }
    walk(t.root);
    expandedNodes = nodes;
    expandedDetails = details;
  }

  function expandAll() {
    expandEverything(tree);
  }

  function collapseAll() {
    expandedNodes = new Set();
    expandedDetails = new Set();
  }
</script>

<div class="page">
  <a
    href="/admin/packages/{ecosystem}/{encodeURIComponent(name)}"
    class="back-link"
  >
    <ArrowLeft class="icon-back" />
    Back to {name}
  </a>

  <div class="page-header">
    <h1 class="page-title">Behavioral Analysis</h1>
    <div class="meta-row">
      <span class="eco-badge">
        {ecosystem}
      </span>
      <span class="meta-name">{name}</span>
      <span class="meta-version">v{version}</span>
    </div>
  </div>

  {#if loading}
    <div class="loading-container">
      <LoaderCircle class="spinner" />
    </div>
  {:else if error}
    <div class="alert alert-error">
      <div class="alert-content">
        <CircleAlert class="icon-alert" />
        <span>{error}</span>
      </div>
    </div>
  {:else if dedupedTree}
    <!-- View mode tabs -->
    <div class="view-tabs">
      <button
        onclick={() => switchView("deduped")}
        class="tab-button"
        class:tab-active={activeView === "deduped"}
      >
        Deduped
      </button>
      <button
        onclick={() => switchView("raw")}
        class="tab-button"
        class:tab-active={activeView === "raw"}
      >
        Raw
      </button>
    </div>

    {#if activeView === "raw" && rawLoading}
      <div class="loading-container">
        <LoaderCircle class="spinner" />
      </div>
    {:else if activeView === "raw" && rawError}
      <div class="alert alert-error">
        <div class="alert-content">
          <CircleAlert class="icon-alert" />
          <span>{rawError}</span>
        </div>
      </div>
    {:else if tree && treeIsEmpty}
      {#if activeView === "deduped"}
        <div class="banner banner-safe">
          <div class="banner-content">
            <ShieldCheck class="icon-shield-safe" />
            <div>
              <h2 class="banner-title safe-title">
                No novel behaviors detected
              </h2>
              <p class="banner-text safe-text">
                All observed activity matches the safe baseline. This package
                version appears to behave normally.
              </p>
            </div>
          </div>
        </div>
      {:else}
        <div class="banner banner-neutral">
          <p class="banner-text neutral-text">No behaviors recorded.</p>
        </div>
      {/if}
    {:else if tree && totalCounts}
      {#if activeView === "deduped"}
        <div class="banner banner-warning">
          <div class="banner-content">
            <ShieldAlert class="icon-shield-warning" />
            <div class="banner-body">
              <span class="banner-count warning-count"
                >{totalBehaviors} novel behavior{totalBehaviors !== 1
                  ? "s"
                  : ""} detected</span
              >
              {@render behaviorSummaryBadges(totalCounts)}
            </div>
          </div>
        </div>
      {:else}
        <div class="banner banner-neutral">
          <div class="banner-content">
            <div class="banner-body">
              <span class="banner-count neutral-count"
                >{totalBehaviors} total behavior{totalBehaviors !== 1
                  ? "s"
                  : ""}</span
              >
              {@render behaviorSummaryBadges(totalCounts)}
            </div>
          </div>
        </div>
      {/if}

      <!-- Controls -->
      <div class="tree-controls">
        <button onclick={expandAll} class="control-button"> Expand All </button>
        <button onclick={collapseAll} class="control-button">
          Collapse All
        </button>
      </div>

      <!-- Process Tree -->
      <div class="tree-panel">
        <div class="tree-header">
          <h3 class="tree-heading">Process Tree</h3>
        </div>
        <div class="tree-body">
          {#each tree.root.children ?? [] as child}
            {@render processNode(child, 0)}
          {/each}
        </div>
      </div>
    {/if}
  {/if}

  <!-- Review Panel -->
  {#if !loading && dedupedTree}
    <div class="review-panel">
      <div class="review-header">
        <MessageSquare class="review-header-icon" />
        <h2 class="review-heading">Manual Review</h2>
        {#if reviewStatus?.manually_approved !== null && reviewStatus?.manually_approved !== undefined}
          {#if reviewStatus.manually_approved}
            <span class="review-status-badge review-status-approved">
              <CheckCircle2 class="review-status-icon" />
              Approved
            </span>
          {:else}
            <span class="review-status-badge review-status-rejected">
              <XCircle class="review-status-icon" />
              Rejected
            </span>
          {/if}
        {:else}
          <span class="review-status-badge review-status-pending">
            <Clock class="review-status-icon" />
            Pending Review
          </span>
        {/if}
      </div>

      {#if reviewLoading}
        <div class="review-loading">
          <LoaderCircle class="spinner-sm" />
          <span>Loading review status...</span>
        </div>
      {:else if reviewError}
        <div class="review-alert review-alert-error">
          {reviewError}
        </div>
      {:else}
        {#if reviewStatus?.review_comment}
          <div class="existing-comment">
            <p class="existing-comment-label">Current comment:</p>
            <p class="existing-comment-text">{reviewStatus.review_comment}</p>
          </div>
        {/if}

        <div class="review-form">
          <div class="review-decision">
            <label class="radio-label">
              <input
                type="radio"
                name="review-decision"
                value="approve"
                checked={reviewApproved}
                onchange={() => (reviewApproved = true)}
              />
              <span class="radio-text radio-approve">Approve</span>
            </label>
            <label class="radio-label">
              <input
                type="radio"
                name="review-decision"
                value="reject"
                checked={!reviewApproved}
                onchange={() => (reviewApproved = false)}
              />
              <span class="radio-text radio-reject">Reject</span>
            </label>
          </div>

          <div class="review-comment-field">
            <label class="review-form-label" for="reviewComment">
              Comment <span class="required">*</span>
            </label>
            <textarea
              id="reviewComment"
              bind:value={reviewComment}
              placeholder="Explain the review decision..."
              rows="3"
              class="review-textarea"
            ></textarea>
          </div>

          {#if submitError}
            <div class="review-alert review-alert-error">
              {submitError}
            </div>
          {/if}

          {#if submitSuccess}
            <div class="review-alert review-alert-success">
              {submitSuccess}
            </div>
          {/if}

          <button
            onclick={handleSubmitReview}
            disabled={submitting}
            class="review-submit"
            class:submit-approve={reviewApproved}
            class:submit-reject={!reviewApproved}
          >
            {#if submitting}
              <LoaderCircle class="spinner-sm" />
            {/if}
            {reviewApproved ? "Approve Version" : "Reject Version"}
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>

{#snippet behaviorSummaryBadges(counts: {
  execs: number;
  files: number;
  connections: number;
  dns: number;
})}
  <div class="badge-row">
    {#if counts.execs > 0}
      <span class="behavior-badge badge-exec">
        <Terminal class="badge-icon" />
        {counts.execs} exec{counts.execs !== 1 ? "s" : ""}
      </span>
    {/if}
    {#if counts.files > 0}
      <span class="behavior-badge badge-file">
        <FileText class="badge-icon" />
        {counts.files} file{counts.files !== 1 ? "s" : ""}
      </span>
    {/if}
    {#if counts.connections > 0}
      <span class="behavior-badge badge-conn">
        <Wifi class="badge-icon" />
        {counts.connections} connection{counts.connections !== 1 ? "s" : ""}
      </span>
    {/if}
    {#if counts.dns > 0}
      <span class="behavior-badge badge-dns">
        <Globe class="badge-icon" />
        {counts.dns} DNS quer{counts.dns !== 1 ? "ies" : "y"}
      </span>
    {/if}
  </div>
{/snippet}

{#snippet processNode(node: ProcessNode, depth: number)}
  {@const hasChildren = (node.children?.length ?? 0) > 0}
  {@const behaviorCount = nodeBehaviorCount(node.behaviors)}
  {@const isExpanded = expandedNodes.has(node.identity)}
  {@const showDetails = expandedDetails.has(node.identity)}
  <div class="tree-node" style="padding-left: {depth * 20}px">
    <!-- Node row -->
    <div class="node-row">
      {#if hasChildren}
        <button onclick={() => toggleNode(node.identity)} class="toggle-button">
          {#if isExpanded}
            <ChevronDown class="toggle-icon" />
          {:else}
            <ChevronRight class="toggle-icon" />
          {/if}
        </button>
      {:else}
        <div class="toggle-spacer"></div>
      {/if}

      <!-- Process name -->
      <button
        onclick={() => {
          if (behaviorCount > 0) toggleDetails(node.identity);
        }}
        class="process-button"
        class:clickable={behaviorCount > 0}
      >
        <span class="process-name">{node.name}</span>

        {#if (node.behaviors.execs?.length ?? 0) > 0}
          <span class="inline-badge inline-exec">
            <Terminal class="inline-icon" />
            {node.behaviors.execs?.length}
          </span>
        {/if}
        {#if (node.behaviors.files_opened?.length ?? 0) > 0}
          <span class="inline-badge inline-file">
            <FileText class="inline-icon" />
            {node.behaviors.files_opened?.length}
          </span>
        {/if}
        {#if (node.behaviors.connections?.length ?? 0) > 0}
          <span class="inline-badge inline-conn">
            <Wifi class="inline-icon" />
            {node.behaviors.connections?.length}
          </span>
        {/if}
        {#if (node.behaviors.dns_queries?.length ?? 0) > 0}
          <span class="inline-badge inline-dns">
            <Globe class="inline-icon" />
            {node.behaviors.dns_queries?.length}
          </span>
        {/if}
      </button>
    </div>

    <!-- Behavior details (expanded) -->
    {#if showDetails && behaviorCount > 0}
      <div class="details-panel">
        {#if (node.behaviors.execs?.length ?? 0) > 0}
          <div class="detail-section">
            <h4 class="detail-heading heading-exec">
              <Terminal class="detail-heading-icon" /> Exec Calls
            </h4>
            <div class="detail-list">
              {#each node.behaviors.execs ?? [] as exec}
                <div class="detail-mono">
                  {exec.argv.join(" ")}
                </div>
              {/each}
            </div>
          </div>
        {/if}

        {#if (node.behaviors.connections?.length ?? 0) > 0}
          <div class="detail-section">
            <h4 class="detail-heading heading-conn">
              <Wifi class="detail-heading-icon" /> Connections
            </h4>
            <div class="detail-list">
              {#each node.behaviors.connections ?? [] as conn}
                <div class="detail-mono">
                  <span class:imds-addr={conn.addr === "169.254.169.254"}>
                    {conn.addr}:{conn.port}
                  </span>
                  {#if conn.addr === "169.254.169.254"}
                    <span class="imds-tag"
                      ><a
                        href="https://securitylabs.datadoghq.com/cloud-security-atlas/attacks/stealing-ec2-instance-role-credentials/"
                        >IMDS</a
                      ></span
                    >
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}

        {#if (node.behaviors.dns_queries?.length ?? 0) > 0}
          <div class="detail-section">
            <h4 class="detail-heading heading-dns">
              <Globe class="detail-heading-icon" /> DNS Queries
            </h4>
            <div class="detail-list">
              {#each node.behaviors.dns_queries ?? [] as query}
                <div class="detail-mono">
                  {query}
                </div>
              {/each}
            </div>
          </div>
        {/if}

        {#if (node.behaviors.files_opened?.length ?? 0) > 0}
          <div class="detail-section-last">
            <h4 class="detail-heading heading-file">
              <FileText class="detail-heading-icon" /> Files Opened ({node
                .behaviors.files_opened?.length})
            </h4>
            <div class="file-list">
              {#each node.behaviors.files_opened ?? [] as file}
                <div class="file-entry">
                  {file}
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Children (expanded) -->
    {#if isExpanded && hasChildren}
      {#each node.children ?? [] as child}
        {@render processNode(child, depth + 1)}
      {/each}
    {/if}
  </div>
{/snippet}

<style>
  .page {
    padding: 2rem;
  }

  .back-link {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    margin-bottom: 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    text-decoration: none;
    transition: color 0.15s;
  }

  .back-link:hover {
    color: var(--text-primary);
  }

  .back-link :global(.icon-back) {
    width: 1rem;
    height: 1rem;
  }

  .page-header {
    margin-bottom: 1.5rem;
  }

  .page-title {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .meta-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-top: 0.25rem;
  }

  .eco-badge {
    display: inline-block;
    padding: 0.25rem 0.5rem;
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    border-radius: 999px;
    background: var(--bg-secondary);
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .meta-name {
    font-weight: 500;
    color: var(--text-primary);
  }

  .meta-version {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    color: var(--text-secondary);
  }

  /* Loading */
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

  /* Alerts */
  .alert {
    margin-bottom: 1rem;
    padding: 1rem;
    border-radius: 8px;
    border: 1px solid;
  }

  .alert-error {
    border-color: rgba(220, 38, 38, 0.2);
    background: rgba(220, 38, 38, 0.08);
  }

  .alert-content {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: #dc2626;
  }

  .alert :global(.icon-alert) {
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
  }

  /* View tabs */
  .view-tabs {
    display: inline-flex;
    gap: 0.25rem;
    margin-bottom: 1rem;
    padding: 0.25rem;
    border-radius: 8px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
  }

  .tab-button {
    padding: 0.375rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    transition:
      background 0.15s,
      color 0.15s,
      box-shadow 0.15s;
  }

  .tab-button:hover {
    color: var(--text-primary);
  }

  .tab-active {
    background: var(--card-bg);
    color: var(--text-primary);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  /* Banners */
  .banner {
    margin-bottom: 1rem;
    padding: 1.25rem;
    border-radius: 8px;
    border: 1px solid;
  }

  .banner-safe {
    border-color: rgba(22, 163, 74, 0.2);
    background: rgba(22, 163, 74, 0.08);
  }

  .banner-warning {
    border-color: rgba(217, 119, 6, 0.2);
    background: rgba(217, 119, 6, 0.08);
  }

  .banner-neutral {
    border-color: var(--border);
    background: var(--bg-secondary);
  }

  .banner-content {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .banner-body {
    flex: 1;
  }

  .banner :global(.icon-shield-safe) {
    width: 2rem;
    height: 2rem;
    color: #16a34a;
    flex-shrink: 0;
  }

  .banner :global(.icon-shield-warning) {
    width: 1.5rem;
    height: 1.5rem;
    color: #d97706;
    flex-shrink: 0;
  }

  .banner-title {
    font-size: 1.125rem;
    font-weight: 600;
  }

  .safe-title {
    color: #15803d;
  }

  .banner-text {
    font-size: 0.875rem;
  }

  .safe-text {
    color: #166534;
  }

  .neutral-text {
    color: var(--text-secondary);
  }

  .banner-count {
    font-weight: 600;
  }

  .warning-count {
    color: #92400e;
  }

  .neutral-count {
    color: var(--text-primary);
  }

  /* Behavior summary badges */
  .badge-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-top: 0.25rem;
  }

  .behavior-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.125rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 500;
    border-radius: 999px;
  }

  .badge-exec {
    background: rgba(139, 92, 246, 0.15);
    color: #6d28d9;
  }

  .badge-file {
    background: rgba(37, 99, 235, 0.15);
    color: #1d4ed8;
  }

  .badge-conn {
    background: rgba(234, 88, 12, 0.15);
    color: #c2410c;
  }

  .badge-dns {
    background: rgba(5, 150, 105, 0.15);
    color: #047857;
  }

  .behavior-badge :global(.badge-icon) {
    width: 0.75rem;
    height: 0.75rem;
  }

  /* Tree controls */
  .tree-controls {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }

  .control-button {
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .control-button:hover {
    background: var(--bg-primary);
  }

  /* Process tree panel */
  .tree-panel {
    border-radius: 8px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  .tree-header {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
    background: var(--bg-secondary);
    border-radius: 8px 8px 0 0;
  }

  .tree-heading {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .tree-body {
    padding: 0.5rem;
  }

  /* Process nodes */
  .node-row {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    transition: background 0.1s;
  }

  .node-row:hover {
    background: var(--bg-secondary);
  }

  .toggle-button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0;
  }

  .toggle-button:hover {
    color: var(--text-primary);
  }

  .toggle-button :global(.toggle-icon) {
    width: 1rem;
    height: 1rem;
  }

  .toggle-spacer {
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
  }

  .process-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    border: none;
    background: transparent;
    padding: 0;
    cursor: default;
    color: inherit;
    font: inherit;
  }

  .process-button.clickable {
    cursor: pointer;
  }

  .process-name {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  /* Inline node badges */
  .inline-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.125rem;
    padding: 0.125rem 0.375rem;
    font-size: 0.7rem;
    border-radius: 999px;
  }

  .inline-exec {
    background: rgba(139, 92, 246, 0.15);
    color: #6d28d9;
  }

  .inline-file {
    background: rgba(37, 99, 235, 0.15);
    color: #1d4ed8;
  }

  .inline-conn {
    background: rgba(234, 88, 12, 0.15);
    color: #c2410c;
  }

  .inline-dns {
    background: rgba(5, 150, 105, 0.15);
    color: #047857;
  }

  .inline-badge :global(.inline-icon) {
    width: 0.625rem;
    height: 0.625rem;
  }

  /* Behavior details panel */
  .details-panel {
    margin: 0 0 0.5rem 1.75rem;
    padding: 0.75rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
  }

  .detail-section {
    margin-bottom: 0.75rem;
  }

  .detail-section-last {
    margin-bottom: 0;
  }

  .detail-heading {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    margin-bottom: 0.25rem;
    font-size: 0.7rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .heading-exec {
    color: #6d28d9;
  }

  .heading-conn {
    color: #c2410c;
  }

  .heading-dns {
    color: #047857;
  }

  .heading-file {
    color: #1d4ed8;
  }

  .detail-heading :global(.detail-heading-icon) {
    width: 0.75rem;
    height: 0.75rem;
  }

  .detail-list {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    max-height: 12rem;
    overflow-y: auto;
  }

  .detail-mono {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.75rem;
    color: var(--text-primary);
    opacity: 0.85;
  }

  .imds-addr {
    font-weight: 700;
    color: #dc2626;
  }

  .imds-tag {
    margin-left: 0.25rem;
    padding: 0.125rem 0.25rem;
    font-size: 0.7rem;
    font-weight: 500;
    border-radius: 4px;
    background: rgba(220, 38, 38, 0.15);
    color: #dc2626;
  }

  .imds-tag a {
    color: inherit;
    text-decoration: none;
  }

  .imds-tag a:hover {
    text-decoration: underline;
  }

  .file-list {
    max-height: 12rem;
    overflow-y: auto;
  }

  .file-entry {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.75rem;
    line-height: 1.25rem;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Review Panel */
  .review-panel {
    margin-top: 2rem;
    border-radius: 8px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    padding: 1.25rem;
  }

  .review-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .review-header :global(.review-header-icon) {
    width: 1.25rem;
    height: 1.25rem;
    color: var(--text-secondary);
  }

  .review-heading {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
    flex: 1;
  }

  .review-status-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.25rem 0.625rem;
    font-size: 0.75rem;
    font-weight: 600;
    border-radius: 999px;
  }

  .review-status-badge :global(.review-status-icon) {
    width: 0.875rem;
    height: 0.875rem;
  }

  .review-status-approved {
    background: rgba(22, 163, 74, 0.12);
    color: #15803d;
  }

  .review-status-rejected {
    background: rgba(220, 38, 38, 0.12);
    color: #dc2626;
  }

  .review-status-pending {
    background: rgba(234, 179, 8, 0.12);
    color: #a16207;
  }

  .review-loading {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 1rem 0;
    color: var(--text-secondary);
    font-size: 0.875rem;
  }

  .review-loading :global(.spinner-sm) {
    width: 1rem;
    height: 1rem;
    animation: spin 1s linear infinite;
  }

  .existing-comment {
    margin-bottom: 1rem;
    padding: 0.75rem;
    border-radius: 6px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
  }

  .existing-comment-label {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 0.25rem;
  }

  .existing-comment-text {
    font-size: 0.875rem;
    color: var(--text-primary);
    white-space: pre-wrap;
  }

  .review-form {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .review-decision {
    display: flex;
    gap: 1rem;
  }

  .radio-label {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    cursor: pointer;
  }

  .radio-text {
    font-size: 0.875rem;
    font-weight: 500;
  }

  .radio-approve {
    color: #15803d;
  }

  .radio-reject {
    color: #dc2626;
  }

  .review-comment-field {
    display: flex;
    flex-direction: column;
  }

  .review-form-label {
    margin-bottom: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .required {
    color: #dc2626;
  }

  .review-textarea {
    width: 100%;
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    font-family: inherit;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    resize: vertical;
    outline: none;
    transition: border-color 0.15s;
  }

  .review-textarea::placeholder {
    color: var(--text-secondary);
  }

  .review-textarea:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px rgba(29, 78, 216, 0.15);
  }

  .review-alert {
    padding: 0.625rem 0.75rem;
    border-radius: 6px;
    font-size: 0.875rem;
  }

  .review-alert-error {
    background: rgba(220, 38, 38, 0.08);
    color: #dc2626;
  }

  .review-alert-success {
    background: rgba(22, 163, 74, 0.08);
    color: #16a34a;
  }

  .review-submit {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    align-self: flex-start;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: #fff;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .review-submit:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .review-submit :global(.spinner-sm) {
    width: 1rem;
    height: 1rem;
    animation: spin 1s linear infinite;
  }

  .submit-approve {
    background: #16a34a;
  }

  .submit-approve:hover:not(:disabled) {
    background: #15803d;
  }

  .submit-reject {
    background: #dc2626;
  }

  .submit-reject:hover:not(:disabled) {
    background: #b91c1c;
  }
</style>
