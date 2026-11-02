<script lang="ts">
  import type { PageData } from './$types';
  import Icon from '#lib/components/Icon.svelte';
  import CopyAddress from '#lib/components/CopyAddress.svelte';
  import StatusBadge from '#lib/components/StatusBadge.svelte';
  import UptimeBadge from '#lib/components/UptimeBadge.svelte';
  import HistoryChart from '#lib/components/HistoryChart.svelte';
  import Markdown from '#lib/components/Markdown.svelte';
  import Modal from '#lib/components/Modal.svelte';
  import { address,iconURL,httpURL,timeLabel } from '#lib/presentation.ts';
  import { onMount } from 'svelte';
  export let data: PageData;
  let embedOpen=false;
  let now=Date.now();
  onMount(() => {const timer=setInterval(() => now=Date.now(),60000);return () => clearInterval(timer);});
  $: server=data.server;
  $: status=server.statuses[0];
  $: peak=server.graph.data.reduce((n,p)=>Math.max(n,p.onlineCount),0);
  $: links=[{label:'Вебсайт',icon:'website',url:httpURL(server.link)},{label:'Telegram',icon:'telegram',url:server.telegram?'https://t.me/'+encodeURIComponent(server.telegram):null},{label:'Discord',icon:'discord',url:server.discord?'https://discord.gg/'+encodeURIComponent(server.discord):null},{label:'YouTube',icon:'youtube',url:httpURL(server.youtube)}].filter(link=>link.url);
</script>
<svelte:head><title>{server.name} — UAMonitor</title><meta name="description" content={'Онлайн та історія гравців Minecraft-сервера '+server.name} /></svelte:head>
<a href="/" class="back-link"><Icon name="arrow" size={16} />До списку серверів</a>
<section class="profile-heading"><div class="profile-identity"><img src={iconURL(server.icon)} alt="" width="72" height="72" /><div><div class="profile-title"><h1>{server.name}</h1><StatusBadge {status} {now} /></div><p class="muted">Minecraft Java Edition</p></div></div><CopyAddress value={address(server)} /></section>
<div class="profile-stats"><div class="panel stat-panel"><Icon name="users" /><span>Гравців онлайн</span><strong>{status?.isOnline ? status.onlineCount ?? 0 : '—'}<small> / {status?.maxOnline ?? '—'}</small></strong></div><div class="panel stat-panel"><Icon name="chart" /><span>Пік за добу</span><strong>{server.graph.data.length?peak:'—'}</strong></div><div class="panel stat-panel"><Icon name="server" /><span>Версія сервера</span><strong class="stat-version" title={status?.version || ''}>{status?.version || 'Ще невідома'}</strong></div><div class="panel stat-panel"><Icon name="clock" /><span>Остання перевірка</span><strong class="stat-time">{status ? timeLabel(status.date,true) : 'Очікуємо'}</strong></div></div>
<div class="profile-grid"><div class="profile-main"><section class="panel"><div class="panel-heading"><h2>Активність гравців</h2><span class="tag">Останні 24 години</span></div><HistoryChart data={server.graph.data} /></section><section class="panel description-panel"><h2>Про сервер</h2><Markdown md={server.description || 'Власник поки не додав опис сервера.'} /></section></div>
<aside class="profile-sidebar"><section class="panel"><h2>Доступність сервера</h2><UptimeBadge uptime={server.uptime} detailed /><p class="small muted">Пропуски перевірок не враховуються як простій. Оцінка залежить від частоти перевірок; короткі збої можуть бути непомітні.</p></section><section class="panel"><h2>Приєднатися до гри</h2><p class="muted">Скопіюйте адресу та додайте сервер у Minecraft Java Edition.</p><CopyAddress value={address(server)} /><p class="small muted">Мережева перевірка виконується кожні 10 хвилин за стандартних налаштувань.</p></section>
{#if links.length}<section class="panel"><h2>Спільнота</h2><div class="community-links">{#each links as link}<a href={link.url!} target="_blank" rel="noopener noreferrer"><span class="community-link-label"><span class="community-service-icon" data-service={link.icon} aria-hidden="true"></span>{link.label}</span><Icon name="external" size={15} /></a>{/each}</div></section>{/if}
<section class="panel"><Icon name="code" size={22} /><h2>Моніторинг на вашому сайті</h2><p class="muted">Додайте картку з актуальним станом і графіком онлайн.</p><button class="button full-width" on:click={()=>embedOpen=true}>Отримати код</button></section></aside></div>
<Modal bind:open={embedOpen} title="Додати моніторинг на сайт" server_id={server.id}><p class="muted">Скопіюйте iframe-код у HTML вашої сторінки.</p></Modal>
