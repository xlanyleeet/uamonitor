<script lang="ts">
  import { onMount } from 'svelte';
  import { refreshAll } from '$app/navigation';
  let busy = false;
  let failed = false;
  async function refresh() {
    if (busy || document.hidden) return;
    busy = true;
    try { await refreshAll(); failed = false; } catch { failed = true; }
    finally { busy = false; }
  }
  onMount(() => { const timer = setInterval(refresh,60000); document.addEventListener('visibilitychange',refresh);
    return () => { clearInterval(timer); document.removeEventListener('visibilitychange',refresh); }; });
</script>
{#if failed}<div class="notice" role="status">Не вдалося оновити дані. Повторимо автоматично.</div>{/if}
