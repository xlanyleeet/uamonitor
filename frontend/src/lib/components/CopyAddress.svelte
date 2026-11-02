<script lang="ts">
  import Icon from './Icon.svelte';
  export let value: string;
  export let compact = false;
  let message = '';
  async function copy() {
    try { await navigator.clipboard.writeText(value); message = 'Адресу скопійовано'; }
    catch { message = 'Скопіюйте адресу: ' + value; }
  }
</script>
<span class="copy-wrap">
  <button class:icon-button={compact} class:button={!compact} class="copy-button" title="Копіювати адресу" aria-label={'Копіювати адресу ' + value} on:click={copy}>
    <Icon name={message === 'Адресу скопійовано' ? 'check' : 'copy'} size={16} />
    {#if !compact}<span class="break-all">{value}</span>{/if}
  </button>
  <span role="status" class:copy-feedback={!!message} class:sr-only={!message}>{message}</span>
</span>
