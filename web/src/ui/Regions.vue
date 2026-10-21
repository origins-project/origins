<script setup lang="ts">
// The parts of a picture somebody is in, outlined over it, each named under its box
// where it is given a name; and one being drawn, dashed. Laid over whatever holds the
// picture, which it fills.
export interface Region {
  key: string
  region: readonly number[]
  label?: string
}
defineProps<{ regions: readonly Region[]; drawing?: readonly number[]; labelled?: boolean }>()

const placed = (r: readonly number[]) => ({
  left: `${r[0]! * 100}%`,
  top: `${r[1]! * 100}%`,
  width: `${(r[2]! - r[0]!) * 100}%`,
  height: `${(r[3]! - r[1]!) * 100}%`,
})
</script>

<template>
  <span class="regions" aria-hidden="true">
    <span v-for="r in regions" :key="r.key" class="region" :style="placed(r.region)" data-region>
      <span v-if="labelled && r.label" class="label">{{ r.label }}</span>
    </span>
    <span v-if="drawing && drawing.length === 4" class="region drawing" :style="placed(drawing)" />
  </span>
</template>

<style scoped>
.regions {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.region {
  position: absolute;
  box-sizing: border-box;
  border: var(--border-strong) solid color-mix(in srgb, var(--surface) 85%, transparent);
  border-radius: var(--radius-sm);
  box-shadow: 0 0 0 var(--border) color-mix(in srgb, var(--ink) 35%, transparent);
}
.drawing {
  border-style: dashed;
  border-color: var(--act);
}
.label {
  position: absolute;
  left: calc(var(--border-strong) * -1);
  top: 100%;
  margin-top: var(--space-1);
  padding: 0 var(--space-1);
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--surface) 92%, transparent);
  color: var(--ink);
  font-size: var(--text-xs);
  white-space: nowrap;
}
</style>
