<script lang="ts">
  import type { ParsedServer } from '#lib/types.ts';
  import { address,iconURL } from '#lib/presentation.ts';
  import CopyAddress from './CopyAddress.svelte';
  import StatusBadge from './StatusBadge.svelte';
  import HistoryChart from './HistoryChart.svelte';
  import { onMount } from 'svelte';
  export let server: ParsedServer;
  export let embed=false;
  let now=Date.now();
  onMount(() => {const timer=setInterval(() => now=Date.now(),60000);return () => clearInterval(timer);});
</script>
<article class="embed-card"><div class="embed-heading"><img src={iconURL(server.icon)} alt="" width="48" height="48" /><div><a href={'/server/'+server.id} target={embed?'_blank':undefined} rel="noopener noreferrer" class="embed-title">{server.name}</a><StatusBadge status={server.statuses[0]} {now} /></div><div class="embed-players">{server.statuses[0]?.onlineCount ?? '—'}<span> / {server.statuses[0]?.maxOnline ?? '—'}</span></div></div><div class="embed-address"><CopyAddress value={address(server)} /><span class="small muted">{server.statuses[0]?.version || 'Minecraft Java'}</span></div><HistoryChart data={server.graph.data} compact /><a class="embed-credit" href="/" target="_blank" rel="noopener noreferrer">UAMonitor ↗</a></article>
