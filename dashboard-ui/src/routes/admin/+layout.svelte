<script lang="ts">
  import type { Snippet } from "svelte";
  import favicon from "$lib/assets/favicon.svg";
  import { Package, ListTodo, FolderKanban, Shield } from "lucide-svelte";
  import { page } from "$app/stores";
  import "../../app.css";

  interface Props {
    children: Snippet;
  }

  const { children }: Props = $props();

  const navItems = [
    { href: "/admin", label: "Packages", icon: Package },
    { href: "/admin/tasks", label: "Tasks", icon: ListTodo },
    { href: "/admin/projects", label: "Projects", icon: FolderKanban },
  ];

  const currentPath = $derived($page.url.pathname);
</script>

<svelte:head>
  <link rel="icon" href={favicon} />
  <title>SPR Admin Dashboard</title>
</svelte:head>

<div class="admin-shell">
  <aside class="sidebar">
    <div class="sidebar-header">
      <div class="brand">
        <Shield class="brand-icon" />
        <div>
          <h1 class="brand-name">SPR</h1>
          <p class="brand-sub">Admin Dashboard</p>
        </div>
      </div>
    </div>
    <nav class="nav">
      {#each navItems as item}
        {@const Icon = item.icon}
        {@const isActive =
          currentPath === item.href ||
          (item.href !== "/admin" && currentPath.startsWith(item.href))}
        <a href={item.href} class="nav-link" class:active={isActive}>
          <Icon class="nav-icon" />
          {item.label}
        </a>
      {/each}
    </nav>
    <div class="sidebar-footer">
      <a href="/" class="back-link">← Back to Search</a>
    </div>
  </aside>
  <main class="content">
    {@render children()}
  </main>
</div>

<style>
  .admin-shell {
    display: flex;
    /* fill remaining viewport below the sticky header */
    height: calc(100vh - 3.5rem);
    background: var(--bg-secondary, #f8fafc);
  }

  .sidebar {
    width: 16rem;
    border-right: 1px solid var(--border, #e2e8f0);
    background: var(--bg-primary, #fff);
    display: flex;
    flex-direction: column;
  }

  .sidebar-header {
    padding: 1.5rem;
    border-bottom: 1px solid var(--border, #e2e8f0);
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .brand :global(.brand-icon) {
    height: 2rem;
    width: 2rem;
    color: var(--accent, #1d4ed8);
  }

  .brand-name {
    font-weight: 600;
    color: var(--text-primary, #0f172a);
    font-size: 1rem;
    line-height: 1.4;
  }

  .brand-sub {
    font-size: 0.75rem;
    color: var(--text-secondary, #64748b);
  }

  .nav {
    flex: 1;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .nav-link {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    border-radius: 6px;
    padding: 0.5rem 0.75rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary, #64748b);
    text-decoration: none;
    transition:
      background 0.15s,
      color 0.15s;
  }

  .nav-link:hover {
    background: var(--bg-secondary, #f1f5f9);
    color: var(--text-primary, #0f172a);
  }

  .nav-link.active {
    background: var(--bg-secondary, #f1f5f9);
    color: var(--text-primary, #0f172a);
  }

  .nav-link :global(.nav-icon) {
    height: 1.25rem;
    width: 1.25rem;
  }

  .sidebar-footer {
    border-top: 1px solid var(--border, #e2e8f0);
    padding: 1rem;
  }

  .back-link {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.875rem;
    color: var(--text-secondary, #64748b);
    text-decoration: none;
    transition: color 0.15s;
  }

  .back-link:hover {
    color: var(--text-primary, #0f172a);
  }

  .content {
    flex: 1;
    overflow: auto;
  }
</style>
