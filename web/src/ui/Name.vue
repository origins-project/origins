<script setup lang="ts">
// A name in its parts, the one the person was called by marked and a nickname in
// quotation marks.
export interface NamePart {
  text: string
  call?: boolean
  nickname?: boolean
  soft?: boolean
}
defineProps<{ parts: readonly NamePart[] }>()
</script>

<template>
  <span class="name" dir="auto">
    <template v-for="(part, i) in parts" :key="i">
      <template v-if="i > 0">{{ ' ' }}</template>
      <span :class="{ call: part.call, nickname: part.nickname, soft: part.soft }">{{ part.text }}</span>
    </template>
  </span>
</template>

<style scoped>
.call {
  text-decoration: underline;
  text-decoration-thickness: var(--border-strong);
  text-underline-offset: 0.14em;
  text-decoration-color: color-mix(in srgb, var(--accent) 55%, transparent);
}
.nickname::before {
  content: '“';
}
.nickname::after {
  content: '”';
}
.soft {
  color: var(--muted);
}
</style>
