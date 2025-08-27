<script lang="ts">
import TaskList from '$lib/TaskList.svelte';
import { onMount } from 'svelte';
import { config } from '$lib/config';
let tasks = [];
let loading = true;
onMount(async () => {
	loading = true;
	const res = await fetch('/api/tasks/json');
	if (res.ok) {
		tasks = await res.json();

		// Initialize config columns and default sort
		if (tasks.length > 0) {
			const keys = Object.keys(tasks[0]);
			config.set({ columns: keys, sort: { key: keys[0], direction: 'asc' } });
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
	<TaskList {tasks} />
{/if}
