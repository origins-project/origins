<script setup lang="ts">
import { Theme } from '../ui'
import Specimens from './ui/Specimens.vue'
</script>

<template>
  <Theme theme="light" data-testid="light"><Specimens theme="light" /></Theme>
  <Theme theme="dark" data-testid="dark"><Specimens theme="dark" /></Theme>
</template>
