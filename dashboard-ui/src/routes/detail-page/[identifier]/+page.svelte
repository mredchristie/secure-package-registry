<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { onMount } from "svelte";

  // Redirect old /detail-page/:identifier?ecosystem=...&version=... to new route
  onMount(() => {
    const identifier = decodeURIComponent(page.params.identifier ?? "");
    const ecosystem = page.url.searchParams.get("ecosystem") ?? "npm";
    const version = page.url.searchParams.get("version") ?? "latest";

    goto(
      `/packages/${ecosystem}/${encodeURIComponent(identifier)}/${version}`,
      { replaceState: true },
    );
  });
</script>

<main class="redirect">Redirecting…</main>

<style>
  .redirect {
    display: flex;
    min-height: 60vh;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
  }
</style>
