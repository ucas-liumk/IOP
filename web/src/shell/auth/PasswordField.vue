<template>
  <div class="pw-wrap">
    <input
      :type="show ? 'text' : 'password'"
      :value="modelValue"
      :placeholder="placeholder"
      :autocomplete="autocomplete"
      :minlength="minlength"
      :required="required"
      :autofocus="autofocus"
      class="input pw-input"
      @input="onInput"
    />
    <button type="button" class="pw-toggle" :aria-label="show ? '隐藏密码' : '显示密码'" @click="show = !show">
      <!-- eye / eye-off -->
      <svg v-if="!show" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
           stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
        <circle cx="12" cy="12" r="3"/>
      </svg>
      <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"
           stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
        <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
        <path d="M14.12 14.12a3 3 0 1 1-4.24-4.24"/>
        <line x1="1" y1="1" x2="23" y2="23"/>
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

withDefaults(defineProps<{
  modelValue: string;
  placeholder?: string;
  autocomplete?: string;
  minlength?: number;
  required?: boolean;
  autofocus?: boolean;
}>(), {
  placeholder: "",
  autocomplete: "current-password",
  minlength: undefined,
  required: false,
  autofocus: false,
});

const emit = defineEmits<{ (e: "update:modelValue", v: string): void }>();

const show = ref(false);

function onInput(e: Event) {
  emit("update:modelValue", (e.target as HTMLInputElement).value);
}
</script>

<style scoped>
.pw-wrap {
  position: relative;
  display: block;
}
.pw-input {
  width: 100%;
  padding-right: 36px;
}
.pw-toggle {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 26px;
  height: 26px;
  border: 0;
  background: transparent;
  cursor: pointer;
  color: var(--text-3);
  display: grid;
  place-items: center;
  border-radius: 5px;
  transition: color 0.15s, background 0.15s;
}
.pw-toggle:hover {
  color: var(--text);
  background: var(--surface-2);
}
.pw-toggle:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 1px;
}
</style>
