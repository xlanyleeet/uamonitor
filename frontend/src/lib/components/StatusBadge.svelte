<script lang="ts">
  import { serverState, stateLabel } from '#lib/presentation.ts';
  import type { Status } from '#lib/types.ts';
  export let status: Status | undefined;
  export let now = Date.now();
  $: state = serverState(status, now);
</script>
<span class="status-badge" data-state={state}><span class="status-dot"></span>{stateLabel[state]}</span>
