<script lang="ts">
  import { onDestroy, tick } from 'svelte';
  import { browser } from '$app/env';
  import Icon from './Icon.svelte';
  export let open = false;
  export let title: string;
  export let server_id = 0;
  let dialog: HTMLDialogElement;
  $: origin = browser ? window.location.origin : '';
  $: code = '<iframe title="UAMonitor" width="500" height="280" src="' + origin + '/s/' + server_id + '?theme=dark"></iframe>';
  async function sync(isOpen: boolean) {
    await tick();
    if (!dialog) return;
    if (isOpen && !dialog.open) dialog.showModal();
    else if (!isOpen && dialog.open) dialog.close();
  }
  $: if (browser) void sync(open);
  onDestroy(() => { if (dialog?.open) dialog.close(); });
</script>
<dialog bind:this={dialog} on:close={() => open = false} on:cancel={() => open = false}
  aria-label={title} class="modal">
  {#if open}
    <div class="modal-heading">
      <h2>{title}</h2>
      <button class="icon-button" aria-label="Закрити" on:click={() => open = false}><Icon name="close" /></button>
    </div>
    <slot />
    <pre class="embed-code">{code}</pre>
    <p class="small muted">Для світлої теми змініть theme=dark на theme=light.</p>
  {/if}
</dialog>
