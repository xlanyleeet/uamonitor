<script lang="ts">
  import '../../app.css';
  import '../../artlist.css';
  import AutoRefresh from '#lib/components/AutoRefresh.svelte';
  import { theme } from '#lib/stores.ts';
  import { browser } from '$app/env';
  $: if (browser) document.documentElement.classList.toggle('dark', $theme === 'dark');
</script>
<div class="embed-shell"><AutoRefresh /><slot /></div>
