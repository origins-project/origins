<script setup lang="ts">
// A word or a number set apart: living, open, private, a count, a grade.
withDefaults(
  defineProps<{ tone?: 'neutral' | 'good' | 'warn' | 'danger' | 'suppose' | 'accent' | 'faint'; mono?: boolean; struck?: boolean }>(),
  { tone: 'neutral' },
)
</script>

<template>
  <span class="badge" :class="[tone, { mono, struck }]"><slot /></span>
</template>

<style scoped>
.badge {
  display: inline-block;
  font-size: var(--text-2xs);
  line-height: var(--leading);
  padding: 0 var(--space-2);
  border-radius: var(--radius-sm);
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
.mono {
  font-family: var(--mono);
  font-weight: var(--weight-bold);
  letter-spacing: 0.04em;
  padding: 0 var(--space-1);
}
.struck {
  text-decoration: line-through;
}
.neutral {
  background: var(--surface-2);
  color: var(--muted);
}
.faint {
  background: var(--surface-2);
  color: var(--faint);
}
.good {
  background: var(--good-soft);
  color: var(--good);
}
.warn {
  background: var(--warn-soft);
  color: var(--warn);
}
.danger {
  background: var(--danger-soft);
  color: var(--danger);
}
.suppose {
  background: var(--suppose-soft);
  color: var(--suppose);
}
.accent {
  background: var(--act-soft);
  color: var(--act);
}
</style>
