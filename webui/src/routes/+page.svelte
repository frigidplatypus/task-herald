<script lang="ts">
import TaskList from '$lib/TaskList.svelte';
import { onMount } from 'svelte';
import { config } from '$lib/config';
import { get } from 'svelte/store';
let tasks = [];
let loading = true;
// Handle refresh event from TaskList to update tasks immediately
function handleRefresh(event: CustomEvent<any>) {
	tasks = event.detail;
}
onMount(async () => {
	loading = true;
	const res = await fetch('/api/tasks/json');
	if (res.ok) {
		tasks = await res.json();

		// If no preferences yet, initialize config from task keys
		const state = get(config);
		if (!state.loading && state.config.columns.length === 0 && tasks.length > 0) {
			const keys = Object.keys(tasks[0]);
			config.set({ loading: false, config: { columns: keys, sort: { key: keys[0], direction: 'asc' } } });
		}
	} else {
		tasks = [];
	}
	loading = false;
});
</script>

<h1>Task List</h1>
{#if loading}
	<p>Loading...</p>
{:else}
	<TaskList {tasks} on:refresh={handleRefresh} />
{/if}
