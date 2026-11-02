<script lang="ts">
  import DOMPurify from 'dompurify';
  import { marked } from 'marked';
  import { onMount } from 'svelte';
  export let md: string;
  let mounted = false;
  onMount(() => { mounted = true; });
  // Hydrate the same empty HTML as SSR, then sanitize and update after mount.
  $: result = mounted ? DOMPurify.sanitize(marked.parse(md, { async: false }), {
    ALLOWED_TAGS: ['p','br','strong','em','ul','ol','li','blockquote','code','pre','a','h2','h3','hr'],
    ALLOWED_ATTR: ['href','title'], ALLOW_DATA_ATTR: false
  }) : '';
</script>
<div class="prose dark:prose-invert break-words">{@html result}</div>
