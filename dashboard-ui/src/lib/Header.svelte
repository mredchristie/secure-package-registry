<script lang="ts">
  // import { browser } from '$app/environment';
  // import { isLoggedIn } from './utils/auth';

  export let isDark = true;
  export let onToggleTheme: () => void;
  export let isLoggedIn: boolean = false;

  import { PUBLIC_HOME_BASE_URL } from "$env/static/public";
</script>

<header>
  <div class="header-left">
    <div class="logo">
      <a class="clear-a-stylings" href="{PUBLIC_HOME_BASE_URL}/">SPR</a>
    </div>
    <a href="/docs" class="nav-link">Docs</a>
    <a href="/pricing" class="nav-link">Pricing</a>
  </div>
  <div class="header-middle">
    <form class="search-bar" method="get" action="/search">
      <input
        type="text"
        name="searchbar"
        placeholder="Search Packages..."
        class=""
      />
      <button type="submit" class="search-button"> Search </button>
    </form>
  </div>
  {#if isLoggedIn}
    <div class="header-right">
      <button class="theme-toggle" on:click={onToggleTheme}>
        {isDark ? "Light" : "Dark"}
      </button>
      <a href="/login" class="navbar-button"> Log Out </a>
    </div>
  {:else}
    <div class="header-right">
      <button class="theme-toggle" on:click={onToggleTheme}>
        {isDark ? "Light" : "Dark"}
      </button>
      <a href="/login" class="navbar-button"> Log In</a>
      <a href="/signup" class="navbar-button"> Sign Up </a>
    </div>
  {/if}
</header>

<style>
  header {
    padding: 0.75rem 2.5rem;
    display: flex;
    /* justify-content: space-between; */
    align-items: center;
    position: sticky;
    top: 0;
    background: var(--bg-primary);
    border-bottom: 1px solid var(--border);
    z-index: 100;
    backdrop-filter: blur(10px);
  }

  .search-button {
    border-top-right-radius: 6px;
    border-bottom-right-radius: 6px;
    background: var(--accent);
    color: var(--bg-primary);
    text-decoration: none;
    padding: 0.5rem 1.75rem;
    font-size: 0.95rem;
    font-weight: 500;
    transition: background 0.2s;
    font-size: 0.875rem;
    font-weight: 500;
    transition: color 0.3s;
  }

  .search-bar {
    width: 100%;
    display: flex;
    justify-content: center;
    margin-left: 2rem;
    margin-right: 2rem;
  }

  .search-bar input {
    padding: 0.5rem;
    font-size: 0.95rem;
    font-weight: 500;
    transition: background 0.2s;
    width: 75%;
    color: var(--text-secondary);
    background: var(--bg-primary);
    accent-color: var(--text-secondary);
    border-top-left-radius: 6px;
    border-bottom-left-radius: 6px;
    border-right: none;
  }

  .search-bar input::placeholder {
    color: var(--text-secondary);
  }

  .search-bar input:focus {
    outline: none;
  }

  .navbar-button {
    background: var(--accent);
    color: var(--bg-primary);
    text-decoration: none;
    padding: 0.5rem 1.75rem;
    border-radius: 6px;
    font-size: 0.95rem;
    font-weight: 500;
    transition: background 0.2s;
    font-size: 0.875rem;
    font-weight: 500;
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

  .header-middle {
    flex: 1;
    display: flex;
    justify-content: center;
  }

  .theme-toggle {
    background: transparent;
    border: none;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--wb-bg);

    background: var(--wb-bg-invert);
    text-decoration: none;
    padding: 0.5rem;
    border-radius: 6px;
    font-size: 0.95rem;
    font-weight: 500;
    transition: background 0.2s;
    font-size: 0.875rem;
    font-weight: 500;
    transition: color 0.3s;
    cursor: pointer;
  }

  /* .theme-toggle:hover { */
  /*        color:  */
  /* } */

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
