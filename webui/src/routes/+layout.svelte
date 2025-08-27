<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import ConfigModal from '$lib/ConfigModal.svelte';
	import { onMount } from 'svelte';
	import { config } from '$lib/config';

	let showConfig = false;

	// Load persisted config from localStorage on client
	onMount(() => {
		const stored = localStorage.getItem('taskherald-config');
		if (stored) {
			try {
				config.set(JSON.parse(stored));
			} catch {}
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<style>
	nav.menu {
		margin-bottom:2em;
		display:flex;
		gap:2em;
		align-items:center;
		background:#f8f9fa;
		padding:1em 2em;
		border-radius:8px;
		box-shadow:0 2px 8px #0001;
		font-size:1.1em;
	}
	nav.menu a {
		text-decoration:none;
		color:#222;
		font-weight:500;
		padding:0.3em 1em;
		border-radius:5px;
		transition:background 0.2s;
	}
	nav.menu a:hover, nav.menu a:focus {
		background:#e2e6ea;
		outline:none;
	}
</style>
<nav class="menu">
		<a href="/">Tasks</a>
		<button type="button"
			style="background:none;border:none;padding:0.3em 1em;font:inherit;color:#222;font-weight:500;border-radius:5px;cursor:pointer;transition:background 0.2s;"
			class="config-btn"
			on:click={() => showConfig = true}
		>Config</button>
</nav>
<ConfigModal show={showConfig} on:close={() => showConfig = false} />
<slot />
