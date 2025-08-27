<script lang="ts">
import { onMount } from 'svelte';

// Default columns available from backend
const availableColumns = [
  { key: 'id', label: 'ID' },
  { key: 'description', label: 'Description' },
  { key: 'project', label: 'Project' },
  { key: 'tags', label: 'Tags' },
  { key: 'status', label: 'Status' },
  { key: 'notification_date', label: 'Notification Date' },
  { key: 'priority', label: 'Priority' },
];

let selectedColumns = [
  'id',
  'description',
  'project',
  'tags',
  'status',
  'notification_date',
  'priority',
];

function moveColumn(from: number, to: number) {
  if (to < 0 || to >= selectedColumns.length) return;
  const col = selectedColumns.splice(from, 1)[0];
  selectedColumns.splice(to, 0, col);
}

function toggleColumn(key: string) {
  const idx = selectedColumns.indexOf(key);
  if (idx === -1) {
    selectedColumns.push(key);
  } else {
    selectedColumns.splice(idx, 1);
  }
}
</script>

<h2>Configure Task List Columns</h2>
<ul>
  {#each availableColumns as col, i}
    <li style="margin-bottom:0.5em">
  <input type="checkbox" checked={selectedColumns.includes(col.key)} on:change={() => toggleColumn(col.key)} />
      {col.label}
      {#if selectedColumns.includes(col.key)}
        <button on:click={() => moveColumn(selectedColumns.indexOf(col.key), selectedColumns.indexOf(col.key)-1)} disabled={selectedColumns.indexOf(col.key) === 0}>↑</button>
        <button on:click={() => moveColumn(selectedColumns.indexOf(col.key), selectedColumns.indexOf(col.key)+1)} disabled={selectedColumns.indexOf(col.key) === selectedColumns.length-1}>↓</button>
      {/if}
    </li>
  {/each}
</ul>

<p>Selected order: {selectedColumns.map(key => availableColumns.find(c => c.key === key)?.label).join(', ')}</p>
