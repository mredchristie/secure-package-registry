<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "$lib/utils";

  interface DialogProps {
    open?: boolean;
    class?: string;
    children: Snippet;
  }

  let { open = false, class: className, children }: DialogProps = $props();

  function close() {
    open = false;
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 bg-black/50" onclick={close}>
    <div
      class={cn(
        "fixed left-[50%] top-[50%] z-50 translate-x-[-50%] translate-y-[-50%] rounded-lg border bg-white shadow-lg",
        className,
      )}
      onclick={(e) => e.stopPropagation()}
    >
      {@render children()}
    </div>
  </div>
{/if}
