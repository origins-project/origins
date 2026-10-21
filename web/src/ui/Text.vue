<script setup lang="ts">
// Words drawn as something other than plain text: muted, faint, small, set in the
// serif or the fixed face, or a label in capitals over what it names.
withDefaults(
  defineProps<{
    tone?: 'ink' | 'muted' | 'faint' | 'act' | 'warn' | 'danger' | 'good' | 'suppose'
    size?: '2xs' | 'xs' | 'sm' | 'md' | 'base' | 'lg' | 'xl'
    face?: 'sans' | 'serif' | 'mono'
    caps?: boolean
    italic?: boolean
    strong?: boolean
    as?: string
  }>(),
  { as: 'span' },
)
</script>

<template>
  <component
    :is="as"
    class="text"
    :class="[tone && `tone-${tone}`, size && `size-${size}`, face && `face-${face}`, { caps, italic, strong }]"
  >
    <slot />
  </component>
</template>

<style scoped>
.text {
  margin: 0;
}
.tone-ink {
  color: var(--ink);
}
.tone-muted {
  color: var(--muted);
}
.tone-faint {
  color: var(--faint);
}
.tone-act {
  color: var(--act);
}
.tone-warn {
  color: var(--warn);
}
.tone-danger {
  color: var(--danger);
}
.tone-good {
  color: var(--good);
}
.tone-suppose {
  color: var(--suppose);
}
.size-2xs {
  font-size: var(--text-2xs);
}
.size-xs {
  font-size: var(--text-xs);
}
.size-sm {
  font-size: var(--text-sm);
}
.size-md {
  font-size: var(--text-md);
}
.size-base {
  font-size: var(--text-base);
}
.size-lg {
  font-size: var(--text-lg);
}
.size-xl {
  font-size: var(--text-xl);
}
.face-sans {
  font-family: var(--sans);
}
.face-serif {
  font-family: var(--serif);
}
.face-mono {
  font-family: var(--mono);
}
.caps {
  font-size: var(--text-2xs);
  font-weight: var(--weight-bold);
  letter-spacing: var(--tracking-caps);
  text-transform: uppercase;
  color: var(--faint);
}
.italic {
  font-style: italic;
}
.strong {
  font-weight: var(--weight-bold);
}
</style>
