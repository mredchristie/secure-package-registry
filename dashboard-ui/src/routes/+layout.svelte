<script lang="ts">
  // imports kinda self explanitory
  import { onMount } from "svelte";
  import Header from "$lib/Header.svelte";
  import favicon from "$lib/assets/favicon.svg";
  // svelte 5 feature props - children contains all page content.
  let { children } = $props();
  // Theme state = true (dark mode) and = false (light mode)
  let isDark = $state(true);

  // Restore saved theme preference from browser storage
  onMount(() => {
    const saved = localStorage.getItem("isDark");
    if (saved !== null) isDark = saved === "true";
  });

  function toggleTheme() {
    // Toggle theme and persist to localStorage
    isDark = !isDark;
    localStorage.setItem("isDark", isDark.toString());
  }
</script>

<svelte:head>
  <!-- Favicon for browser tab - this was already in the file when I rebased Mohammeds branch so kept it in -->
  <link rel="icon" href={favicon} />
</svelte:head>

<div class="app" class:dark={isDark}>
  <Header {isDark} onToggleTheme={toggleTheme} />
  <!-- Header component with theme toggle (appears on all pages) -->
  {@render children()}
  <!-- Page content rendered here via {@render children()} -->
</div>

<style>
  /*
    CSS custom properties defined here cascade to all child pages.
    Only variables live here — all other styles belong in their own files.

    Color variable reference — light / dark
    --bg-primary:     #fff            / #0a0a0f
    --bg-secondary:   #fafafa         / #1a1a2e
    --text-primary:   #111            / #fff
    --text-secondary: #4b5563         / #ccc
    --accent:         #1d4ed8 (blue)  / #4fc3f7 (sky blue)
    --accent-hover:   #1e40af         / #51cf66 (green)
    --border:         #e5e7eb         / #2a2a3e
    --card-bg:        #fff            / rgba(26,26,46,0.5)
    --card-border:    #e5e7eb         / rgba(79,195,247,0.2)
  */

  /* Light mode */
  .app {
    --bg-primary: #fff;
    --bg-secondary: #fafafa;
    --text-primary: #111;
    --text-secondary: #4b5563;
    --accent: #1d4ed8;
    --accent-hover: #1e40af;
    --border: #e5e7eb;
    --card-bg: #fff;
    --card-border: #e5e7eb;

    font-family:
      system-ui,
      -apple-system,
      sans-serif;
    background: var(--bg-primary);
    color: var(--text-primary);
    line-height: 1.6;
    min-height: 100vh;
    transition:
      background 0.3s,
      color 0.3s;
  }

  /* Dark mode */
  .app.dark {
    --bg-primary: #0a0a0f;
    --bg-secondary: #1a1a2e;
    --text-primary: #fff;
    --text-secondary: #ccc;
    --accent: #4fc3f7;
    --accent-hover: #51cf66;
    --border: #2a2a3e;
    --card-bg: rgba(26, 26, 46, 0.5);
    --card-border: rgba(79, 195, 247, 0.2);
  }

  /* Universal resets — intentionally global */
  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(body) {
    overflow-x: hidden;
    overscroll-behavior: none;
  }
</style>
