<script setup lang="ts">
import { ref } from 'vue'
import { Refused } from '../api/client'
import { useClient } from '../api/use'
import { login } from '../labels'
import { Button, Card, Field, Notice, Page, Stack } from '../ui'

const client = useClient()
const passphrase = ref('')
const refused = ref<string>()
const busy = ref(false)
const insecure = location.protocol === 'http:'

async function signIn() {
  busy.value = true
  refused.value = undefined
  try {
    await client.login(passphrase.value)
    location.reload()
  } catch (e) {
    refused.value = e instanceof Refused ? e.message : String(e)
    busy.value = false
  }
}
</script>

<template>
  <Page narrow>
    <Card>
      <form @submit.prevent="signIn">
        <Stack gap="4">
          <Notice v-if="insecure" tone="warn">{{ login.insecure }}</Notice>
          <Field v-model="passphrase" type="password" :label="login.passphrase" />
          <Notice v-if="refused" tone="danger">{{ refused }}</Notice>
          <Button type="submit" variant="primary" :disabled="busy">{{ login.signIn }}</Button>
        </Stack>
      </form>
    </Card>
  </Page>
</template>
