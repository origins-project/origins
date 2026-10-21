<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { useCorpus } from './api/corpus'
import Frame from './frame/Frame.vue'
import LoginPage from './pages/LoginPage.vue'

// The frame around whichever page the route names, drawn again whenever the view or
// the supposing changes what it reads; or, where the server asks a passphrase first,
// the page that asks it.
const corpus = useCorpus()
const { session, signIn, shown, ready } = corpus
onMounted(() => void corpus.start())
onBeforeUnmount(() => corpus.stop())
</script>

<template>
  <Frame>
    <LoginPage v-if="signIn" />
    <RouterView v-else-if="session && ready" :key="shown" />
  </Frame>
</template>
