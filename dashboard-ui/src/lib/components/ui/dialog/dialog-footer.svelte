<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "$lib/utils";

  interface Props {
    class?: string;
    children: Snippet;
  }

  let { class: className, children }: Props = $props();
</script>

<div class={cn("flex items-center justify-end gap-2 p-6 pt-0", className)}>
  {@render children()}
</div>
