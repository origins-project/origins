<script setup lang="ts">
// One thing in a `Timeline`: its year, given where it differs from the one before, and
// a card of what happened. A period is drawn as a span down the line; a quiet one is
// somebody else's, a relative's birth or death, set back from the person's own.
defineProps<{ year?: string; quiet?: boolean; span?: boolean; path?: string }>()
</script>

<template>
  <li class="moment" :class="{ quiet, span }" :data-moment="path">
    <span class="year">{{ year }}</span>
    <div class="card"><slot /></div>
  </li>
</template>

<style scoped>
.moment {
  position: relative;
  display: grid;
  grid-template-columns: var(--year) minmax(0, 1fr);
  gap: var(--space-6);
  padding: var(--space-2) 0;
}
.moment::before {
  content: '';
  position: absolute;
  left: calc(var(--year) + var(--space-6) / 2 + var(--space-1) + var(--border-strong) / 2 - var(--moment) / 2);
  top: calc(var(--space-2) + var(--space-3));
  width: var(--moment);
  height: var(--moment);
  box-sizing: border-box;
  border-radius: var(--radius-pill);
  background: var(--surface);
  border: var(--border-strong) solid var(--accent);
}
.span::after {
  content: '';
  position: absolute;
  left: calc(var(--year) + var(--space-6) / 2 + var(--space-1) - var(--border));
  top: calc(var(--space-2) + var(--space-3) + var(--moment));
  bottom: 0;
  width: calc(2 * var(--border-strong));
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--accent) 35%, transparent);
}
.year {
  font: var(--weight-medium) var(--text-lg) var(--serif);
  color: var(--muted);
  text-align: right;
  padding-top: var(--space-1);
  font-variant-numeric: tabular-nums;
}
.card {
  min-width: 0;
  background: var(--surface);
  border: var(--border) solid var(--line);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
  box-shadow: var(--shadow);
}
.quiet .card {
  background: transparent;
  box-shadow: none;
  border-style: dashed;
  padding: var(--space-2) var(--space-3);
  color: var(--muted);
  font-size: var(--text-md);
}
.quiet::before {
  border-color: var(--line-strong);
}
@media (max-width: 40em) {
  .moment {
    grid-template-columns: var(--year-phone) minmax(0, 1fr);
    gap: var(--space-5);
  }
  .moment::before {
    left: calc(var(--year-phone) + var(--space-5) / 2 + var(--space-1) + var(--border-strong) / 2 - var(--moment) / 2);
  }
  .span::after {
    left: calc(var(--year-phone) + var(--space-5) / 2 + var(--space-1) - var(--border));
  }
  .year {
    font-size: var(--text-base);
  }
}
</style>
