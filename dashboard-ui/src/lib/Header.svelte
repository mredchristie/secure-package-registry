<script lang="ts">
  import { PUBLIC_HOME_BASE_URL } from "$env/static/public";
  import type { Session } from "$lib/client";
  import { enhance } from "$app/forms";

  interface Props {
    isDark?: boolean;
    onToggleTheme: () => void;
    user?: Session["user"] | null;
  }

  let { isDark = true, onToggleTheme, user = null }: Props = $props();
</script>

<header>
  <div class="header-left">
    <div class="logo">
      <a class="clear-a-stylings" href="{PUBLIC_HOME_BASE_URL}/">SPR</a>
    </div>
  </div>
  <div class="header-middle">
    <a href="/search" class="nav-link">Search</a>
  </div>
  {#if user}
    <div class="header-right">
      <button class="theme-toggle" onclick={onToggleTheme}>
        {isDark ? "Light" : "Dark"}
      </button>
      <a href="/profile" class="nav-link user-link">
        {user.name || "User"}
      </a>
      <form method="POST" action="/logout" use:enhance>
        <button type="submit" class="navbar-button">Log Out</button>
      </form>
    </div>
  {:else}
    <div class="header-right">
      <button class="theme-toggle" onclick={onToggleTheme}>
        {isDark ? "Light" : "Dark"}
      </button>
      <a href="/login" class="navbar-button">Sign In</a>
    </div>
  {/if}
</header>

<style>
  header {
    padding: 0.75rem 2.5rem;
    display: flex;
    align-items: center;
    position: sticky;
    top: 0;
    background: var(--bg-primary);
    border-bottom: 1px solid var(--border);
    z-index: 100;
    backdrop-filter: blur(10px);
  }

  .navbar-button {
    background: var(--accent);
    color: var(--bg-primary);
    text-decoration: none;
    padding: 0.5rem 1.75rem;
    border-radius: 6px;
    font-size: 0.875rem;
    font-weight: 500;
    transition: background 0.2s;
  }

  .navbar-button:hover {
    background: var(--accent-hover);
  }

  form {
    display: contents;
  }

  .clear-a-stylings {
    text-decoration: none;
    color: inherit;
  }

  .logo {
    font-size: 2rem;
    font-weight: 700;
    color: var(--accent);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-left: auto;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .header-middle {
    flex: 1;
    display: flex;
    justify-content: center;
  }

  .theme-toggle {
    background: var(--wb-bg-invert);
    border: none;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--wb-bg);
    text-decoration: none;
    padding: 0.5rem;
    border-radius: 6px;
    transition: background 0.2s;
    cursor: pointer;
  }

  .nav-link {
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 0.875rem;
    font-weight: 500;
    transition: color 0.3s;
  }

  .nav-link:hover {
    color: var(--accent);
  }

  .user-link {
    font-weight: 600;
    color: var(--text-primary);
  }
</style>
