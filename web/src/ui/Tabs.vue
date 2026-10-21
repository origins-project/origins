<script setup lang="ts">
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from 'reka-ui'
import type { Option } from './Select.vue'

// A page's tabs, each drawn by the slot of its value; one disabled has nothing to show
// yet.
const model = defineModel<string>()
defineProps<{ tabs: Option[]; label: string }>()
</script>

<template>
  <TabsRoot v-model="model" :default-value="tabs[0]?.value" class="tabs">
    <TabsList class="list" :aria-label="label">
      <TabsTrigger v-for="tab in tabs" :key="tab.value" :value="tab.value" :disabled="tab.disabled" class="trigger">
        {{ tab.label }}
      </TabsTrigger>
    </TabsList>
    <TabsContent v-for="tab in tabs" :key="tab.value" :value="tab.value" class="content">
      <slot :name="tab.value" />
    </TabsContent>
  </TabsRoot>
</template>

<style scoped>
.list {
  display: flex;
  gap: var(--space-1);
  border-bottom: var(--border) solid var(--line);
  margin-bottom: var(--space-6);
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
}
.trigger {
  border: 0;
  background: none;
  padding: var(--space-3) var(--space-3);
  color: var(--muted);
  cursor: pointer;
  font-size: var(--text-base);
  border-bottom: var(--border-strong) solid transparent;
  margin-bottom: calc(-1 * var(--border));
  white-space: nowrap;
}
.trigger:hover {
  color: var(--ink);
}
.trigger:disabled {
  color: var(--faint);
  cursor: default;
}
.trigger[data-state='active'] {
  color: var(--ink);
  border-bottom-color: var(--act);
  font-weight: var(--weight-medium);
}
.trigger:focus-visible {
  outline-offset: calc(-1 * var(--border-strong));
}
.content:focus-visible {
  outline: none;
}
</style>
