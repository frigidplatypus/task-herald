<script lang="ts">

  import { config } from '$lib/config';
  import { derived } from 'svelte/store';
  import { datepicker } from '$lib/actions/datepicker';
  import { createEventDispatcher } from 'svelte';
  // Pull in flatpickr styles for calendar popover
  import 'flatpickr/dist/flatpickr.min.css';

  export let tasks: Array<Record<string, any>> = [];
  // Event dispatcher to notify parent of updates
  const dispatcher = createEventDispatcher();

  // Wait for config to load before rendering
  let currentConfig = { columns: [], sort: { key: '', direction: 'asc' } };
  let loading = true;
  config.subscribe((state) => {
    currentConfig = state.config;
    loading = state.loading;
  });

  // Format date strings for display
  function formatDate(dateStr: string | undefined): string {
    if (!dateStr) return '';
    let d: Date;
    // Handle TaskWarrior UTC format: YYYYMMDDTHHMMSSZ
    const m = dateStr.match(/^(\d{4})(\d{2})(\d{2})T(\d{2})(\d{2})(\d{2})Z$/);
    if (m) {
      const [ , y, mo, da, h, mi, s ] = m;
      d = new Date(Date.UTC(+y, +mo - 1, +da, +h, +mi, +s));
    } else if (dateStr.includes('T')) {
      d = new Date(dateStr);
    } else {
      d = new Date(dateStr.replace(' ', 'T'));
    }
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
  }

  // Render a cell based on key, formatting dates and special types
  function renderCell(task: Record<string, any>, key: string): string {
    // Date fields: format using localized date/time
    const dateKeys = ['due', 'modified', 'start', 'end', 'wait', 'entry'];
    if (dateKeys.includes(key)) {
      return formatDate(task[key]);
    }
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
    // Fallback to raw value
    return task[key] ?? '';
  }


  // Apply sorting based on configuration, with date fields compared as dates
  let displayedTasks: typeof tasks = tasks;
  $: {
    const key = currentConfig?.sort?.key;
    const dir = currentConfig?.sort?.direction;
    if (key) {
      const dateKeys = ['due', 'modified', 'start', 'end', 'wait', 'entry', 'notification_date'];
      displayedTasks = [...tasks].sort((a, b) => {
        let v1 = a[key] ?? '';
        let v2 = b[key] ?? '';
        // Compare date fields by timestamp
        if (dateKeys.includes(key)) {
          const d1 = new Date(v1 as string);
          const d2 = new Date(v2 as string);
          const t1 = isNaN(d1.getTime()) ? 0 : d1.getTime();
          const t2 = isNaN(d2.getTime()) ? 0 : d2.getTime();
          if (t1 < t2) return dir === 'asc' ? -1 : 1;
          if (t1 > t2) return dir === 'asc' ? 1 : -1;
          return 0;
        }
        // Fallback to string comparison
        v1 = String(v1);
        v2 = String(v2);
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


  // Inline edit of any date field
  function startEdit(task: Record<string, any>, key: string) {
    editingId = task.id;
    editingKey = key;
    editValue = toDatetimeLocal(task[key]);
  editDate = editValue ? new Date(editValue) : null;
  }

  async function saveEdit(task: Record<string, any>) {
    if (!editingKey) return;
    // Send key and value to backend
    const body: any = { uuid: task.uuid };
    body[editingKey] = editValue;
     const res = await fetch('/api/tasks/set-notification-date', {
       method: 'POST',
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify(body)
     });
     if (res.ok) {
      task[editingKey] = editValue ? editValue.replace('T', ' ') : '';
     }
    editingId = null;
    editingKey = null;
   }
  // Editable date fields
  const dateFields = ['notification_date', 'due', 'start', 'wait'];
  // Popover modal for date fields
  let showDatePicker = false;
  let pickerTask: Record<string, any> | null = null;
  let pickerKey = '';
  let pickerValue = '';

  // Reference to the hidden date input and modal container for appendTo
  let pickerInput: HTMLInputElement;
  let modalElement: HTMLDivElement;

  function openDatePicker(task: Record<string, any>, key: string) {
    showDatePicker = true;
    pickerTask = task;
    pickerKey = key;
    const raw = task[key];
    pickerValue = raw ? raw.replace(' ', 'T') : '';
  }
  function cancelDatePicker() {
    showDatePicker = false;
    pickerTask = null;
  }
  async function confirmDatePicker() {
  if (!pickerTask) return;
  const body: any = { uuid: pickerTask.uuid };
  body[pickerKey] = pickerValue;
  const res = await fetch('/api/tasks/modify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (res.ok) {
    const updatedTasks: any[] = await res.json();
    tasks = updatedTasks;
    // notify parent to refresh tasks
    dispatcher('refresh', updatedTasks);
  }
  showDatePicker = false;
}
  // Clear a date field immediately
  async function clearDateField(task: Record<string, any>, key: string) {
    // set value to empty and submit
    pickerTask = task;
    pickerKey = key;
    pickerValue = '';
    await confirmDatePicker();
  }
</script>

  {#if loading}
    <div>Loading preferences...</div>
  {:else}
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
                {#if dateFields.includes(key)}
     <div class="date-cell">
       <!-- date display button -->
       <button
         class="open-btn"
         type="button"
         aria-label="Set date"
         on:click={() => openDatePicker(task, key)}
       >{renderCell(task, key) || '—'}</button>
       <!-- clear only if there's a value -->
       {#if task[key]}
         <button
           class="clear-btn"
           type="button"
           on:click|stopPropagation={() => clearDateField(task, key)}
           aria-label="Clear date"
         >×</button>
       {/if}
     </div>
   {:else}
     {renderCell(task, key)}
   {/if}
               </td>
             {/each}
          </tr>
      {/each}
      </tbody>
    </table>
  {/if}

  {#if showDatePicker}
    <div class="modal-backdrop" on:click={cancelDatePicker}></div>
  <div class="modal" bind:this={modalElement}>
      <!-- Render flatpickr calendar inline inside modal -->
      <input
        bind:this={pickerInput}
        type="text"
        bind:value={pickerValue}
        use:datepicker={{
          enableTime: true,
          dateFormat: 'Y-m-d H:i',
          defaultDate: pickerValue,
          inline: true,
          appendTo: modalElement,
          onChange: (_dates, dateStr) => { pickerValue = dateStr; }
        }}
        style="position:absolute; opacity:0; width:1px; height:1px; border:none; padding:0; margin:0;"
      />
      <div class="modal-actions">
        <button type="button" on:click={confirmDatePicker}>OK</button>
        <button type="button" on:click={cancelDatePicker}>Cancel</button>
      </div>
    </div>
  {/if}

<style>
  .modal-backdrop {
    position: fixed; inset: 0; background: rgba(0,0,0,0.3); z-index: 10;
  }
  .modal { position: fixed; top:50%; left:50%; transform: translate(-50%,-50%); background: #fff; color: #222; padding:1em; border-radius:4px; z-index:11; width:300px; }
  .modal-actions { display:flex; justify-content:flex-end; gap:0.5em; margin-top:1em; }
  .date-cell { display: flex; align-items: center; justify-content: space-between; width: 100%; }
  .clear-btn { background: none; border: none; padding: 0; font-size: 0.9em; cursor: pointer; color: #888; }
  .clear-btn:hover { color: #f00; }
  .open-btn { background: none; border: none; padding: 0; color: inherit; cursor: pointer; }
</style>
