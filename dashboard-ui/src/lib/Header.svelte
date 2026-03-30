<script lang="ts">
  import { PUBLIC_HOME_BASE_URL } from "$env/static/public";
  import { authClient } from "$lib/client";

  interface Props {
    isDark?: boolean;
    onToggleTheme: () => void;
    user?: { name: string; email: string } | null;
  }

  let { isDark = true, onToggleTheme, user = null }: Props = $props();

  async function handleSignOut() {
    await authClient.signOut();
    window.location.href = "/login";
  }
</script>

<header>
  <div class="header-left">
    <div class="logo">
      <a class="clear-a-stylings" href="{PUBLIC_HOME_BASE_URL}/">SPR</a>
    </div>
    <a href="/docs" class="nav-link">Docs</a>
    <a href="/pricing" class="nav-link">Pricing</a>
  </div>
  {#if user}
    <div class="header-right">
      <button class="theme-toggle" onclick={onToggleTheme}>
        {isDark ? "Light" : "Dark"}
      </button>
      <span class="user-name">{user.name}</span>
      <button class="navbar-button" onclick={handleSignOut}>Sign Out</button>
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
    border: none;
    border-radius: 6px;
    font-size: 0.875rem;
    font-weight: 500;
    font-family: inherit;
    cursor: pointer;
    transition: color 0.3s;
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

  .theme-toggle {
    background: var(--wb-bg-invert);
    border: none;
    color: var(--wb-bg);
    text-decoration: none;
    padding: 0.5rem 1.75rem;
    border-radius: 6px;
    font-size: 0.875rem;
    font-weight: 500;
    transition: color 0.3s;
    cursor: pointer;
  }

  .user-name {
    font-size: 0.875rem;
    color: var(--text-secondary);
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
</style>
