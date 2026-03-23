<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    class?: string;
    children: Snippet;
  }

  let { children }: Props = $props();
</script>

<h2 class="text-lg font-semibold leading-none tracking-tight text-slate-900">
  {@render children()}
</h2>
