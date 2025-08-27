<script lang="ts">
  import { config } from '$lib/config';
  import { get } from 'svelte/store';

  export let tasks: Array<Record<string, any>> = [];

  // User configuration from store
  let currentConfig = get(config);
  config.subscribe((c) => (currentConfig = c));

  // Inline edit state
  let editingId: number | null = null;
  let editValue: string = '';

  // Format date strings for display
  function formatDate(dateStr: string | undefined): string {
    if (!dateStr) return '';
    const d = dateStr.includes('T')
      ? new Date(dateStr)
      : new Date(dateStr.replace(' ', 'T'));
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
  }

  // Render a cell based on key
  function renderCell(task: Record<string, any>, key: string): string {
    if (key === 'notification_date') {
      return formatDate(task.notification_date);
    }
    if (key === 'urgency') {
      // display urgency as a number with two decimals
      const val = Number(task.urgency);
      return isNaN(val) ? '' : val.toFixed(2);
    }
    if (key === 'tags' && Array.isArray(task.tags)) {
      return task.tags.join(', ');
    }
    return task[key] ?? '';
  }


  // Apply sorting based on configuration
  let displayedTasks: typeof tasks = tasks;
  $: {
    const key = currentConfig?.sort?.key;
    const dir = currentConfig?.sort?.direction;
    if (key) {
      displayedTasks = [...tasks].sort((a, b) => {
        const v1 = a[key] ?? '';
        const v2 = b[key] ?? '';
        if (v1 < v2) return dir === 'asc' ? -1 : 1;
        if (v1 > v2) return dir === 'asc' ? 1 : -1;
        return 0;
      });
    } else {
      displayedTasks = tasks;
    }
  }
  // Convert to local datetime string for editing
  function toDatetimeLocal(dateStr: string | undefined): string {
    if (!dateStr) return '';
    const d = dateStr.includes('T')
      ? new Date(dateStr)
      : new Date(dateStr.replace(' ', 'T'));
    if (isNaN(d.getTime())) return '';
    const pad = (n: number) => n.toString().padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }

  // Inline edit of notification_date
  function startEdit(task) {
    editingId = task.id;
    editValue = toDatetimeLocal(task.notification_date);
  }

  async function saveEdit(task) {
    const body = { uuid: task.uuid, notification_date: editValue };
    const res = await fetch('/api/tasks/set-notification-date', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (res.ok) {
      task.notification_date = editValue ? editValue.replace('T', ' ') : '';
    }
    editingId = null;
  }
</script>

<table>
  <thead>
    <tr>
      {#each currentConfig.columns as key}
        <th>{key.replace('_', ' ')}</th>
      {/each}
    </tr>
  </thead>
  <tbody>
    {#each displayedTasks as task}
      <tr>
        {#each currentConfig.columns as key}
          <td>
            {#if key === 'notification_date'}
              {#if editingId === task.id}
                <input type="datetime-local" bind:value={editValue}
                  on:blur={() => saveEdit(task)}
                  on:keydown={(e) => e.key === 'Enter' && saveEdit(task)}
                  autofocus
                />
                <button type="button" on:click={() => { editValue = ''; saveEdit(task); }} style="margin-left:0.5em">Clear</button>
              {:else}
                <span style="cursor:pointer" on:click={() => startEdit(task)}>
                  {formatDate(task.notification_date)}
                </span>
              {/if}
            {:else}
              {renderCell(task, key)}
            {/if}
          </td>
        {/each}
      </tr>
    {/each}
  </tbody>
</table>
