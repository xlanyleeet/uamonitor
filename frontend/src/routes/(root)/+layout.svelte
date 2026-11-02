<script lang="ts">
  import '../../app.css';
  import '../../artlist.css';
  import '../../admission.css';
  import { theme } from '#lib/stores.ts';
  import { browser } from '$app/env';
  import Icon from '#lib/components/Icon.svelte';
  import AutoRefresh from '#lib/components/AutoRefresh.svelte';
  $: if (browser) document.documentElement.classList.toggle('dark',$theme === 'dark');
</script>
<svelte:head><title>UAMonitor — сервери Minecraft</title><meta name="description" content="Українські Minecraft-сервери: знайдіть спільноту, перевірте онлайн та історію гравців." /></svelte:head>
<a class="skip-link" href="#main">До вмісту</a>
<header class="site-header">
  <div class="site-container nav-content">
    <a class="brand" href="/" aria-label="UAMonitor — головна"><span class="brand-symbol"><Icon name="server" size={25} /></span><span>UAMonitor<span class="brand-caption"> — сервери Minecraft</span></span></a>
    <nav class="nav-actions" aria-label="Головна навігація">
      <a class="button button-quiet random-link" href="/random" data-sveltekit-reload><Icon name="shuffle" size={16} /><span>Випадковий сервер</span></a>
      <a class="button button-primary" href="/add-server"><Icon name="plus" size={16} /><span>Додати сервер</span></a>
      <button class="icon-button" aria-label={$theme === 'dark' ? 'Увімкнути світлу тему' : 'Увімкнути темну тему'} title="Змінити тему" on:click={() => theme.set($theme === 'dark' ? 'light' : 'dark')}><Icon name={$theme === 'dark' ? 'sun' : 'moon'} /></button>
    </nav>
  </div>
</header>
<main id="main" class="site-container page-content"><AutoRefresh /><slot /></main>
<footer class="site-footer"><div class="site-container footer-content">
  <div><a class="brand footer-brand" href="/"><Icon name="server" size={20} />UAMonitor</a><p>Знайдіть свою спільноту.<br />Грайте на українських серверах Minecraft.</p></div>
  <nav aria-label="Посилання у підвалі"><strong>Проєкт</strong><a href="/">Список серверів</a><a href="/requirements">Вимоги до серверів</a><a href="/add-server">Подати заявку</a><a href="/about">Про моніторинг</a></nav>
  <div class="footer-credit"><strong>Відкритий код</strong><a href="https://github.com/Andrmist/uamonitor" target="_blank" rel="noopener noreferrer">Оригінальний UAMonitor ↗</a><a href="https://github.com/Wojtazzzz/ArtList.dev" target="_blank" rel="noopener noreferrer">Дизайн на основі ArtList.dev ↗</a><a href="/licenses/artlist-MIT.txt">Ліцензія MIT</a></div>
</div><div class="site-container footer-bottom">UAMonitor · Minecraft Java Edition<span>Історія онлайн за останні 24 години</span></div></footer>
