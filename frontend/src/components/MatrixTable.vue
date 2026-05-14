<script setup>
import { computed } from 'vue'

const props = defineProps({
  matrix: { type: Array, default: () => [] },
})

const normalized = computed(() => (Array.isArray(props.matrix) ? props.matrix : []))

function formatCell(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return String(value)
  return n.toFixed(6)
}
</script>

<template>
  <div class="matrix">
    <table v-if="normalized.length" class="matrix-table">
      <tbody>
        <tr v-for="(row, i) in normalized" :key="i">
          <td v-for="(cell, j) in row" :key="j">{{ formatCell(cell) }}</td>
        </tr>
      </tbody>
    </table>
    <p v-else class="muted">Sin datos</p>
  </div>
</template>

<style scoped>
.matrix {
  overflow: auto;
}

.matrix-table {
  border-collapse: collapse;
  width: 100%;
}

.matrix-table td {
  border: 1px solid var(--border);
  padding: 6px 8px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.muted {
  color: var(--muted);
}
</style>

