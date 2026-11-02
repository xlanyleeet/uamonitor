<script lang="ts">
  import type { Server } from '#lib/types.ts';
  export let uptime: Server['uptime'];
  export let detailed = false;
  $: percent = uptime?.percent;
</script>
<div class="uptime-summary" title="Оцінка за останні 30 днів. Пропуски перевірок не вважаються простоєм. Короткі збої між перевірками можуть бути непомітні.">
  <span class="small muted">Аптайм за 30 днів</span>
  <strong>{percent != null ? percent.toLocaleString('uk-UA', { maximumFractionDigits: 2 }) + '%' : 'Немає даних'}</strong>
  {#if detailed && percent != null}<span class="small muted">Покриття даними: {uptime!.coverage.toLocaleString('uk-UA', { maximumFractionDigits: 1 })}% періоду</span>{/if}
</div>
<style>
  .uptime-summary { display: flex; flex-direction: column; gap: .25rem; }
  strong { font-variant-numeric: tabular-nums; }
</style>
