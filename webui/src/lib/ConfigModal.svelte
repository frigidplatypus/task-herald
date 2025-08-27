<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { onMount } from 'svelte';
  import { dndzone } from 'svelte-dnd-action';
  import { config } from '$lib/config';

  // Modal visibility
  export let show: boolean;
  const dispatch = createEventDispatcher();

  // Available TaskWarrior columns fetched from API
  let columns: { key: string; label: string }[] = [];


  import { get } from 'svelte/store';

  // Fetch columns from backend and initialize modal state from config
  onMount(async () => {
    try {
      const res = await fetch('/api/tasks/json');
      const tasks = await res.json();
      if (Array.isArray(tasks) && tasks.length > 0) {
        columns = Object.keys(tasks[0]).map((k) => ({ key: k, label: k }));
        // Include computed urgency field if missing
        if (!columns.find((c) => c.key === 'urgency')) {
          columns.push({ key: 'urgency', label: 'Urgency' });
        }
      }
    } catch (err) {
      console.error('Failed to load columns', err);
    }
    // Initialize drag items and sort from config
    const currentConfig = get(config);
    // If config columns are set, use that order and enabled state
    if (currentConfig.columns && currentConfig.columns.length > 0) {
      dndItems = columns.map((c) => ({
        id: c.key,
        label: c.label,
        enabled: currentConfig.columns.includes(c.key)
      }));
      // Reorder dndItems to match config.columns order for enabled columns
      const enabledOrder = currentConfig.columns;
      dndItems = [
        ...enabledOrder.map(key => dndItems.find(item => item.id === key)).filter(Boolean),
        ...dndItems.filter(item => !enabledOrder.includes(item.id))
      ];
    } else {
      dndItems = columns.map((c) => ({ id: c.key, label: c.label, enabled: true }));
    }
    defaultSortKey = currentConfig.sort?.key || columns[0]?.key || '';
    sortDirection = currentConfig.sort?.direction || 'asc';
  });

  // Items for drag-and-drop: include enabled state (initialized after fetch)
  let dndItems: { id: string; label: string; enabled: boolean }[] = [];

  // Selected keys derived from enabled items
  $: selectedColumns = dndItems.filter((item) => item.enabled).map((item) => item.id);

  // Default sort configuration
  let defaultSortKey = '';
  let sortDirection: 'asc' | 'desc' = 'asc';

  // Toggle enabled state on double-click
  function toggleEnable(id: string) {
    dndItems = dndItems.map((item) =>
      item.id === id ? { ...item, enabled: !item.enabled } : item
    );
  }

  // Close and emit configuration
  function apply() {
    // Save config to store
    config.set({ columns: selectedColumns, sort: { key: defaultSortKey, direction: sortDirection } });
    dispatch('close');
  }

  // Handle DnD events, ignore drops outside zone
  function handleConsider(e) {
    const items = e.detail.items;
    if (items.length === dndItems.length) {
      dndItems = items;
    }
  }
  function handleFinalize(e) {
    const items = e.detail.items;
    if (items.length === dndItems.length) {
      dndItems = items;
    }
  }
</script>

{#if show}
  <div class="modal-backdrop" on:click={apply}></div>
  <div class="modal">
    <h2>Configure Columns & Sort</h2>

    <!-- Reorderable column blocks -->
    <div
      class="blocks"
      use:dndzone={{ items: dndItems, flipDurationMs: 150 }}
      on:consider={handleConsider}
      on:finalize={handleFinalize}
    >
      {#each dndItems as { id, label, enabled } (id)}
        <div
          class="block {enabled ? '' : 'disabled'}"
          data-id={id}
          on:dblclick={() => toggleEnable(id)}
        >
          {label}
        </div>
      {/each}
    </div>

    <!-- Default sort selector -->
    <div class="sort-config">
      <label>Sort by:</label>
      <select bind:value={defaultSortKey}>
        {#each columns as col}
          <option value={col.key}>{col.label}</option>
        {/each}
      </select>
      <label>
        <input type="radio" bind:group={sortDirection} value="asc" /> Ascending
      </label>
      <label>
        <input type="radio" bind:group={sortDirection} value="desc" /> Descending
      </label>
    </div>

    <button type="button" class="apply-btn" on:click={apply}>
      Apply
    </button>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.3);
    z-index: 10;
  }
  .modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    background: #fff;
    color: #222;
    border-radius: 8px;
    padding: 1.5em;
    z-index: 11;
    width: 320px;
  }
  .blocks {
    display: flex;
    flex-direction: column;
    gap: 0.5em;
    margin-bottom: 1em;
  }
  .block {
    padding: 0.5em;
    background: #e0e0e0;
    color: #222;
    border-radius: 4px;
    cursor: grab;
    user-select: none;
    display: flex;
    justify-content: space-between;
  }
  .block.disabled {
    opacity: 0.5;
  }
  .sort-config {
    display: flex;
    align-items: center;
    gap: 0.5em;
    margin-bottom: 1em;
  }
  .apply-btn {
    padding: 0.5em 1em;
    background: #007acc;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }
  .apply-btn:hover {
    background: #005fa3;
  }
</style>
