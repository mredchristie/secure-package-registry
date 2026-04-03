<script lang="ts">
  import { goto } from "$app/navigation";
  import { projectsAPI } from "$lib/api";
  import { AlertCircle, ArrowLeft, Loader2, Upload } from "lucide-svelte";

  let projectName = $state("");
  let fileContent = $state("");
  let fileName = $state("");
  let loading = $state(false);
  let error = $state<string | null>(null);

  function handleFileSelect(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    fileName = file.name;
    const reader = new FileReader();
    reader.onload = (e) => {
      fileContent = (e.target?.result as string) ?? "";

      // Auto-detect project name from the file if not set.
      if (!projectName) {
        try {
          const parsed = JSON.parse(fileContent);
          if (parsed.name) {
            projectName = parsed.name;
          }
        } catch {
          // Not valid JSON or no name field — leave blank.
        }
      }
    };
    reader.readAsText(file);
  }

  async function handleSubmit() {
    if (!projectName.trim()) {
      error = "Project name is required";
      return;
    }
    if (!fileContent) {
      error = "Please select a file";
      return;
    }

    loading = true;
    error = null;

    try {
      const result = await projectsAPI.upload(projectName.trim(), fileContent);
      await goto(`/projects/${result.id}`);
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to create project";
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>New Project - SPR</title>
</svelte:head>

<div class="page">
  <div class="page-header">
    <a href="/projects" class="back-link">
      <ArrowLeft class="icon-sm" />
      Back to Projects
    </a>
  </div>

  <div class="form-card">
    <div class="form-header">
      <h1 class="form-title">New Project</h1>
      <p class="form-subtitle">
        Upload a package.json or package-lock.json to track your dependencies
      </p>
    </div>

    <div class="form-body">
      <div class="form-field">
        <label class="form-label" for="projectName">Project Name</label>
        <input
          id="projectName"
          type="text"
          bind:value={projectName}
          placeholder="e.g., my-app"
          class="form-input"
        />
        <p class="form-hint">
          A unique name for this project. Auto-detected from the uploaded file.
        </p>
      </div>

      <div class="form-field">
        <label class="form-label" for="fileUpload">Lock File</label>
        <div class="file-drop-zone">
          <input
            id="fileUpload"
            type="file"
            accept=".json"
            onchange={handleFileSelect}
            class="file-input"
          />
          <div class="file-drop-content">
            <Upload class="upload-icon" />
            {#if fileName}
              <p class="file-name">{fileName}</p>
              <p class="file-hint">Click to choose a different file</p>
            {:else}
              <p class="file-drop-text">Click to select a file</p>
              <p class="file-hint">package.json or package-lock.json</p>
            {/if}
          </div>
        </div>
      </div>

      {#if error}
        <div class="alert alert-error">
          <AlertCircle class="icon-error" />
          <span>{error}</span>
        </div>
      {/if}
    </div>

    <div class="form-footer">
      <a href="/projects" class="btn-secondary">Cancel</a>
      <button onclick={handleSubmit} disabled={loading} class="btn-primary">
        {#if loading}
          <Loader2 class="icon-spin" />
        {/if}
        Create Project
      </button>
    </div>
  </div>
</div>

<style>
  .page {
    padding: 2rem;
    max-width: 40rem;
    margin: 0 auto;
  }

  .page-header {
    margin-bottom: 1.5rem;
  }

  .back-link {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    text-decoration: none;
    transition: color 0.15s;
  }

  .back-link:hover {
    color: var(--text-primary);
  }

  .back-link :global(.icon-sm) {
    width: 1rem;
    height: 1rem;
  }

  .form-card {
    border-radius: 12px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  }

  .form-header {
    padding: 1.5rem;
    border-bottom: 1px solid var(--border);
  }

  .form-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .form-subtitle {
    margin-top: 0.25rem;
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .form-body {
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .form-field {
    display: flex;
    flex-direction: column;
  }

  .form-label {
    margin-bottom: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .form-input {
    height: 2.5rem;
    width: 100%;
    padding: 0 0.75rem;
    font-size: 0.875rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    outline: none;
    transition: border-color 0.15s;
  }

  .form-input::placeholder {
    color: var(--text-secondary);
  }

  .form-input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px rgba(29, 78, 216, 0.15);
  }

  .form-hint {
    margin-top: 0.375rem;
    font-size: 0.75rem;
    color: var(--text-secondary);
    opacity: 0.8;
  }

  .file-drop-zone {
    position: relative;
    border-radius: 8px;
    border: 2px dashed var(--border);
    padding: 2rem;
    text-align: center;
    cursor: pointer;
    transition:
      border-color 0.15s,
      background 0.15s;
  }

  .file-drop-zone:hover {
    border-color: var(--accent);
    background: rgba(29, 78, 216, 0.03);
  }

  .file-input {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    opacity: 0;
    cursor: pointer;
  }

  .file-drop-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    pointer-events: none;
  }

  .file-drop-content :global(.upload-icon) {
    width: 2rem;
    height: 2rem;
    color: var(--text-secondary);
  }

  .file-drop-text {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .file-name {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--accent);
  }

  .file-hint {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .alert {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem;
    border-radius: 6px;
    font-size: 0.875rem;
  }

  .alert-error {
    background: rgba(220, 38, 38, 0.08);
    color: #dc2626;
  }

  .alert :global(.icon-error) {
    width: 1rem;
    height: 1rem;
    flex-shrink: 0;
  }

  .form-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    padding: 1rem 1.5rem;
    border-top: 1px solid var(--border);
  }

  .btn-secondary {
    display: inline-flex;
    align-items: center;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 6px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    cursor: pointer;
    text-decoration: none;
    transition: background 0.15s;
  }

  .btn-secondary:hover {
    background: var(--bg-primary);
  }

  .btn-primary {
    display: inline-flex;
    align-items: center;
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: #fff;
    background: var(--accent);
    border: none;
    border-radius: 6px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-primary:hover {
    background: var(--accent-hover);
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary :global(.icon-spin) {
    width: 1rem;
    height: 1rem;
    margin-right: 0.5rem;
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
</style>
