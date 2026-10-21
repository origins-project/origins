<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import Portrait from './Portrait.vue'

// Somebody among the people a page lists, a parent, a child, a partner: their picture,
// their name, how they are related where that is said, and their lifespan. Given to, it
// is a link to them; struck, it is a link disproven; ghost, somebody with no record.
const props = defineProps<{
  name: string
  to?: RouteLocationRaw
  lifespan?: string
  relation?: string
  sex?: string
  src?: string
  region?: readonly number[]
  initials?: string
  struck?: boolean
  ghost?: boolean
}>()
const as = computed(() => (props.to !== undefined ? RouterLink : 'span'))
</script>

<template>
  <component :is="as" class="person" :class="{ struck, ghost }" :to="to">
    <Portrait :src="src" :region="region" :initials="initials" :sex="sex" />
    <span class="words">
      <span class="name">{{ name }}</span>
      <span v-if="relation" class="relation">{{ relation }}</span>
      <span v-if="lifespan" class="lifespan">{{ lifespan }}</span>
    </span>
  </component>
</template>

<style scoped>
.person {
  display: flex;
  gap: var(--space-3);
  align-items: center;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius);
  border: var(--border) solid var(--line);
  background: var(--surface);
  color: var(--ink);
  text-decoration: none;
  min-width: 0;
}
a.person:hover {
  border-color: var(--line-strong);
  background: var(--surface-2);
}
a.person:focus-visible {
  outline: var(--border-strong) solid var(--act);
  outline-offset: var(--border);
}
.words {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.name {
  font-weight: var(--weight-medium);
  font-size: var(--text-md);
  line-height: var(--leading-tight);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.relation {
  font-size: var(--text-2xs);
  color: var(--accent);
}
.lifespan {
  font-size: var(--text-xs);
  color: var(--muted);
}
.ghost {
  border-style: dashed;
  background: none;
  color: var(--muted);
}
.struck .name {
  text-decoration: line-through;
  text-decoration-color: var(--danger);
}
.struck {
  opacity: 0.65;
}
</style>
