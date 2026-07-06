<template>
  <div class="input-wrap">
    <label v-if="label" :for="inputId" class="input-label">{{ label }}</label>

    <div :class="['input-field', { 'input-field--error': error, 'input-field--focused': focused }]">
      <slot name="prefix" />

      <input
        :id="inputId"
        :type="currentType"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :autocomplete="autocomplete"
        class="input-el"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        @focus="focused = true"
        @blur="focused = false"
      />

      <!-- Toggle para senha -->
      <button
        v-if="type === 'password'"
        type="button"
        class="input-eye"
        @click="showPassword = !showPassword"
        tabindex="-1"
      >
        <Eye    v-if="!showPassword" :size="15" />
        <EyeOff v-else               :size="15" />
      </button>
    </div>

    <p v-if="error" class="input-error">{{ error }}</p>
    <p v-else-if="hint" class="input-hint">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Eye, EyeOff } from '@lucide/vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  label?:       string
  placeholder?: string
  type?:        string
  error?:       string
  hint?:        string
  disabled?:    boolean
  autocomplete?: string
}>(), { type: 'text', disabled: false })

defineEmits<{ 'update:modelValue': [value: string] }>()

const focused      = ref(false)
const showPassword = ref(false)
const inputId      = `input-${Math.random().toString(36).slice(2)}`
const currentType  = computed(() => props.type === 'password' && showPassword.value ? 'text' : props.type)
</script>

<style scoped>
.input-wrap { display: flex; flex-direction: column; gap: 6px; }

.input-label {
  font-family: var(--mono);
  font-size: 11px;
  font-weight: 500;
  color: var(--text3);
  letter-spacing: 1px;
  text-transform: uppercase;
}

.input-field {
  display: flex;
  align-items: center;
  background: var(--bg3);
  border: 1px solid var(--border2);
  border-radius: 11px;
  transition: border-color 0.2s;
  overflow: hidden;
}

.input-field--focused { border-color: rgba(0,229,255,0.5); }
.input-field--error   { border-color: rgba(239,68,68,0.5); }

.input-el {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  padding: 11px 14px;
  font-family: var(--font);
  font-size: 13px;
  color: var(--text);
  width: 100%;
}
.input-el::placeholder { color: var(--text3); }
.input-el:disabled { opacity: 0.5; cursor: not-allowed; }

.input-eye {
  width: 36px; height: 36px;
  display: flex; align-items: center; justify-content: center;
  background: transparent; border: none; cursor: pointer;
  color: var(--text3); flex-shrink: 0;
  transition: color 0.2s;
}
.input-eye:hover { color: var(--text2); }
.input-eye svg { width: 15px; height: 15px; }

.input-error { font-family: var(--mono); font-size: 10px; color: var(--red); }
.input-hint  { font-family: var(--mono); font-size: 10px; color: var(--text3); }
</style>
