<script lang="ts">
  import type { GraphPointData } from '#lib/types.ts';
  import { timeLabel } from '#lib/presentation.ts';
  export let data: GraphPointData[];
  export let compact = false;
  $: points = [...data].sort((a,b) => Date.parse(a.date)-Date.parse(b.date));
  $: peak = points.reduce((n,p) => Math.max(n,p.onlineCount),0);
  $: max = Math.max(1,peak);
  $: start = points.length ? Date.parse(points[0].date) : 0;
  $: span = points.length > 1 ? Math.max(1,Date.parse(points[points.length-1].date)-start) : 1;
  $: line = points.map(p => ((Date.parse(p.date)-start)/span*720).toFixed(1)+','+(150-p.onlineCount/max*135).toFixed(1)).join(' ');
</script>
<figure class:mini-chart={compact} class:history-chart={!compact} aria-label="Кількість гравців за останні 24 години">
  {#if points.length}
    {#if !compact}<div class="chart-topline"><span>Гравців онлайн</span><span>Пік: <strong>{peak}</strong></span></div>{/if}
    <svg viewBox="0 0 720 160" preserveAspectRatio="none" role="img" aria-label={'Онлайн за добу, пік ' + peak}>
      <title>Історія онлайн за добу. Найбільше гравців: {peak}</title>
      {#if !compact}{#each [15,60,105,150] as y}<line x1="0" y1={y} x2="720" y2={y} class="chart-grid" />{/each}{/if}
      <polygon points={'0,160 '+line+' 720,160'} class="chart-area" />
      <polyline points={line} fill="none" class="chart-line" vector-effect="non-scaling-stroke" />
      {#if points.length === 1}<circle cx="0" cy={150-points[0].onlineCount/max*135} r="3" fill="currentColor" />{/if}
    </svg>
    {#if !compact}<figcaption class="chart-times"><span>{timeLabel(points[0].date,true)}</span><span>{timeLabel(points[points.length-1].date,true)}</span></figcaption>{/if}
  {:else}<div class="chart-empty">{compact ? 'Немає даних' : 'Історія з’явиться після перших перевірок сервера'}</div>{/if}
</figure>
