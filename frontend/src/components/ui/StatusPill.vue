<template>
  <span :class="['status-pill', cls]">
    <span class="status-dot" :class="{ 'status-dot--pulse': status === 'rodando' }" />
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const map: Record<string, { cls: string; label: string }> = {
  // Empresa / Grupo
  ativo:      { cls: 'pill-green',  label: 'Ativo' },
  ativa:      { cls: 'pill-green',  label: 'Ativa' },
  inativo:    { cls: 'pill-gray',   label: 'Inativo' },
  inativa:    { cls: 'pill-gray',   label: 'Inativa' },
  deletando:  { cls: 'pill-red',    label: 'Excluindo' },
  pausado:    { cls: 'pill-yellow', label: 'Pausado' },
  // Sync job
  pendente:   { cls: 'pill-gray',   label: 'Pendente' },
  rodando:    { cls: 'pill-blue',   label: 'Rodando' },
  concluido:  { cls: 'pill-green',  label: 'Concluído' },
  erro:       { cls: 'pill-red',    label: 'Erro' },
  // Status sync empresa
  erro_sync:  { cls: 'pill-red',    label: 'Erro' },
}

const entry   = computed(() => map[props.status] ?? { cls: 'pill-gray', label: props.status })
const cls     = computed(() => entry.value.cls)
const label   = computed(() => entry.value.label)
</script>

<style scoped>
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 9px;
  border-radius: 20px;
  font-family: var(--mono);
  font-size: 10px;
  font-weight: 600;
  white-space: nowrap;
}
.pill-green  { background: rgba(34,197,94,0.12);  color: #22c55e; }
.pill-red    { background: rgba(239,68,68,0.12);   color: #ef4444; }
.pill-blue   { background: rgba(0,229,255,0.10);   color: #00e5ff; }
.pill-yellow { background: rgba(245,158,11,0.12);  color: #f59e0b; }
.pill-gray   { background: rgba(255,255,255,0.06); color: var(--text3); }

.status-dot {
  width: 5px; height: 5px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}
.status-dot--pulse { animation: pulse 2s infinite; }
@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.3} }
</style>
