<script lang="ts">
  import type { PageData } from './$types';
  import ServerCard from '#lib/components/ServerCard.svelte';

  export let data: PageData;
</script>

<svelte:head><title>{data.server.name} — UAMonitor</title></svelte:head>
<ServerCard server={data.server} embed={true} />
