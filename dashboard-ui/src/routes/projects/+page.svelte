<script lang="ts">
  import { projectsAPI } from "$lib/api";
  import type { Project } from "$lib/types/api";
  import { Plus, AlertCircle, Loader2, Trash2 } from "lucide-svelte";

  let projects = $state<Project[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);

  // Delete confirmation
  let deletingId = $state<number | null>(null);
  let deleteError = $state<string | null>(null);

  async function loadProjects() {
    loading = true;
    error = null;
    try {
      const response = await projectsAPI.list();
      projects = response.items;
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load projects";
    } finally {
      loading = false;
    }
  }

  async function deleteProject(id: number) {
    deletingId = id;
    deleteError = null;
    try {
      await projectsAPI.delete(id);
      projects = projects.filter((p) => p.id !== id);
    } catch (e) {
      deleteError = e instanceof Error ? e.message : "Failed to delete project";
    } finally {
      deletingId = null;
    }
  }

  function formatDate(dateStr?: string): string {
    if (!dateStr) return "-";
    try {
      return new Date(dateStr).toLocaleDateString("en-GB", {
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        month: "short",
        year: "numeric",
      });
    } catch {
      return dateStr.slice(0, 19);
    }
  }

  $effect(() => {
    loadProjects();
  });
</script>

<svelte:head>
  <title>Projects - SPR</title>
</svelte:head>

<div class="page">
  <div class="page-header">
    <div>
      <h1 class="page-title">Projects</h1>
      <p class="page-subtitle">Track dependencies across your applications</p>
    </div>
    <a href="/projects/new" class="add-button">
      <Plus class="icon-sm" />
      New Project
    </a>
  </div>

  {#if deleteError}
    <div class="error-banner" style="margin-bottom: 1rem">
      <div class="error-content">
        <AlertCircle class="icon-error" />
        <span>{deleteError}</span>
      </div>
    </div>
  {/if}

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
  {:else if projects.length === 0}
    <div class="empty-state">
      <p class="empty-text">No projects yet</p>
      <p class="empty-hint">
        Upload a package.json or package-lock.json to start tracking
        dependencies
      </p>
    </div>
  {:else}
    <div class="projects-grid">
      {#each projects as project}
        <div class="project-card">
          <a href="/projects/{project.id}" class="project-card-link">
            <div class="card-top">
              <span class="project-name">{project.name}</span>
              <span class="source-badge">{project.source_type}</span>
            </div>
            <div class="card-meta">
              <span class="meta-item">
                Updated {formatDate(project.updated_at)}
              </span>
            </div>
          </a>
          <button
            class="delete-button"
            title="Delete project"
            disabled={deletingId === project.id}
            onclick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              deleteProject(project.id);
            }}
          >
            {#if deletingId === project.id}
              <Loader2 class="icon-xs-spin" />
            {:else}
              <Trash2 class="icon-xs" />
            {/if}
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .page {
    padding: 2rem;
    max-width: 60rem;
    margin: 0 auto;
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
    text-decoration: none;
    transition: background 0.15s;
  }

  .add-button:hover {
    background: var(--accent-hover);
  }

  .add-button :global(.icon-sm) {
    width: 1rem;
    height: 1rem;
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

  .projects-grid {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .project-card {
    position: relative;
    border-radius: 12px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    transition:
      transform 0.15s,
      border-color 0.15s,
      box-shadow 0.15s;
  }

  .project-card:hover {
    transform: translateY(-2px);
    border-color: rgba(29, 78, 216, 0.35);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  }

  .project-card-link {
    display: block;
    padding: 1rem 1.25rem;
    text-decoration: none;
  }

  .card-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
  }

  .project-name {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .project-card:hover .project-name {
    color: var(--accent);
  }

  .source-badge {
    display: inline-block;
    flex-shrink: 0;
    padding: 0.125rem 0.625rem;
    font-size: 0.7rem;
    font-weight: 600;
    border-radius: 999px;
    background: var(--bg-secondary);
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .card-meta {
    margin-top: 0.5rem;
    display: flex;
    gap: 1rem;
  }

  .meta-item {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  .delete-button {
    position: absolute;
    right: 0.75rem;
    bottom: 0.75rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-secondary);
    cursor: pointer;
    opacity: 0;
    transition:
      opacity 0.15s,
      color 0.15s,
      background 0.15s;
  }

  .project-card:hover .delete-button {
    opacity: 1;
  }

  .delete-button:hover {
    color: #dc2626;
    background: rgba(220, 38, 38, 0.08);
    border-color: rgba(220, 38, 38, 0.3);
  }

  .delete-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .delete-button :global(.icon-xs) {
    width: 0.875rem;
    height: 0.875rem;
  }

  .delete-button :global(.icon-xs-spin) {
    width: 0.875rem;
    height: 0.875rem;
    animation: spin 1s linear infinite;
  }
</style>
