<script lang="ts">
  import type { PageData } from './$types';
  import { onMount } from 'svelte';
  import Icon from '#lib/components/Icon.svelte';
  import CopyAddress from '#lib/components/CopyAddress.svelte';
  import StatusBadge from '#lib/components/StatusBadge.svelte';
  import UptimeBadge from '#lib/components/UptimeBadge.svelte';
  import HistoryChart from '#lib/components/HistoryChart.svelte';
  import { filterServers, type Sort } from '#lib/catalog.ts';
  import { address, iconURL, serverState } from '#lib/presentation.ts';
  export let data: PageData;
  let query = '';
  let state = 'all';
  let sort: Sort = 'players-desc';
  let currentPage = 1;
  let now = Date.now();
  const pageSize = 20;
  onMount(() => { const timer=setInterval(() => now=Date.now(),60000); return () => clearInterval(timer); });
  $: online = data.servers.filter(s => serverState(s.statuses[0],now) === 'online');
  $: players = online.reduce((n,s) => n+(s.statuses[0]?.onlineCount ?? 0),0);
  $: filtered = filterServers(data.servers,query,state,sort,now);
  $: pageCount = Math.max(1,Math.ceil(filtered.length/pageSize));
  $: shownPage = Math.min(currentPage,pageCount);
  $: rows = filtered.slice((shownPage-1)*pageSize,shownPage*pageSize);
  function sortColumn(column: 'name' | 'players') { sort = sort === column+'-desc' ? column+'-asc' as Sort : column+'-desc' as Sort; currentPage=1; }
  function reset() { query=''; state='all'; sort='players-desc'; currentPage=1; }
</script>
<section class="catalog-intro" aria-labelledby="catalog-title"><div><p class="eyebrow">УКРАЇНСЬКА MINECRAFT-СПІЛЬНОТА</p><h1 id="catalog-title">Знайдіть свій сервер.</h1><p class="intro-description">Актуальний онлайн, історія гравців і нові місця для гри.</p></div>
  <div class="catalog-totals"><div><strong>{data.servers.length}</strong><span>серверів</span></div><div><strong>{online.length}</strong><span>онлайн</span></div><div><strong>{players.toLocaleString('uk-UA')}</strong><span>гравців</span></div></div>
</section>
<section aria-label="Каталог серверів">
  <div class="catalog-toolbar">
    <label class="search-field"><Icon name="search" size={19} /><span class="sr-only">Пошук за назвою, адресою або версією</span><input type="search" bind:value={query} on:input={() => currentPage=1} placeholder="Пошук за назвою, IP або версією…" /></label>
    <div class="filter-controls"><label><span class="sr-only">Стан сервера</span><select bind:value={state} on:change={() => currentPage=1}><option value="all">Усі сервери</option><option value="online">Онлайн</option><option value="offline">Офлайн</option><option value="stale">Застарілі дані</option><option value="unknown">Ще не перевірені</option></select></label>
      <label><span class="sr-only">Сортування</span><select bind:value={sort} on:change={() => currentPage=1}><option value="players-desc">Найбільше гравців</option><option value="players-asc">Найменше гравців</option><option value="name-asc">Назва: А → Я</option><option value="name-desc">Назва: Я → А</option></select></label></div>
  </div>
  <div class="catalog-meta"><span role="status">{filtered.length === data.servers.length ? 'Усього серверів: ' : 'Знайдено серверів: '}{filtered.length}</span><span class="live-note"><span class="status-dot"></span>Автоматичне оновлення</span></div>
  <div class="server-table-wrap"><table class="server-table"><caption class="sr-only">Список українських Minecraft-серверів</caption>
    <thead><tr><th scope="col" class="rank-cell">#</th><th scope="col" aria-sort={sort === 'name-asc' ? 'ascending' : sort === 'name-desc' ? 'descending' : 'none'}><button on:click={() => sortColumn('name')}>Сервер <Icon name="sort" size={13} /></button></th><th scope="col" aria-sort={sort === 'players-asc' ? 'ascending' : sort === 'players-desc' ? 'descending' : 'none'}><button on:click={() => sortColumn('players')}>Гравці <Icon name="sort" size={13} /></button></th><th scope="col" class="version-cell">Аптайм / версія</th><th scope="col" class="history-cell">Онлайн за добу</th><th scope="col" class="actions-cell">Дії</th></tr></thead>
    <tbody>{#each rows as server,index (server.id)}<tr>
      <td class="rank-cell">{(shownPage-1)*pageSize+index+1}</td>
      <td class="server-name-cell"><a href={'/server/'+server.id} class="server-identity"><img src={iconURL(server.icon)} width="48" height="48" alt="" /><span class="server-identity-text"><strong>{server.name}</strong><span class="server-address">{address(server)}</span><span class="mobile-status"><StatusBadge status={server.statuses[0]} {now} /></span></span></a></td>
      <td class="players-cell"><div class="players-count">{server.statuses[0]?.isOnline ? server.statuses[0].onlineCount ?? 0 : '—'}<span> / {server.statuses[0]?.maxOnline ?? '—'}</span></div><span class="desktop-status"><StatusBadge status={server.statuses[0]} {now} /></span></td>
      <td class="version-cell"><UptimeBadge uptime={server.uptime} /><span class="version-text" title={server.statuses[0]?.version || ''}>{server.statuses[0]?.version || '—'}</span></td>
      <td class="history-cell"><HistoryChart data={server.graph.data} compact /></td>
      <td class="actions-cell"><div class="row-actions"><CopyAddress value={address(server)} compact /><a class="icon-button" href={'/server/'+server.id} aria-label={'Відкрити '+server.name}><Icon name="chevron" size={17} /></a></div></td>
    </tr>{:else}<tr><td colspan="6" class="empty-state"><div class="empty-icon"><Icon name={data.servers.length ? 'search' : 'server'} size={28} /></div><h2>{data.servers.length ? 'Нічого не знайдено' : 'Тут починається ваша спільнота'}</h2><p>{data.servers.length ? 'Спробуйте іншу назву або змініть фільтри.' : 'Каталог поки порожній. Додайте перший сервер, щоб почати моніторинг.'}</p>{#if data.servers.length}<button class="button" on:click={reset}>Скинути фільтри</button>{:else}<a class="button button-primary" href="/about#add-server"><Icon name="plus" size={16} />Як додати сервер</a>{/if}</td></tr>{/each}</tbody>
  </table></div>
  {#if pageCount>1}<nav class="pagination" aria-label="Сторінки каталогу"><button class="button" disabled={shownPage===1} on:click={() => currentPage=shownPage-1}>← Назад</button><span>Сторінка {shownPage} із {pageCount}</span><button class="button" disabled={shownPage===pageCount} on:click={() => currentPage=shownPage+1}>Далі →</button></nav>{/if}
</section>
<section class="catalog-notes"><div><h2>Більше, ніж список адрес</h2><p>Перевірте стан сервера та подивіться, як змінювався онлайн протягом доби. Відкрийте профіль, щоб дізнатися більше про спільноту.</p></div><div><h2>Ваш сервер — у списку</h2><p>Маєте український Minecraft-проєкт? Додайте його до моніторингу, щоб гравцям було простіше вас знайти.</p><a class="text-link" href="/about#add-server">Як додати сервер <span aria-hidden="true">↗</span></a></div></section>
