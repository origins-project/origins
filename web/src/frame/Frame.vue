<script setup lang="ts">
import { computed, ref, watch, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { kindsInOrder } from '../api/client'
import { useCorpus } from '../api/corpus'
import { useClient } from '../api/use'
import ChangeDialog from '../edit/ChangeDialog.vue'
import { app, edit as said, frame, kinds } from '../labels'
import { listed } from '../search'
import {
  Banner,
  Bar,
  Brand,
  Drawer,
  IconButton,
  Live,
  Menu,
  MenuItem,
  MenuSeparator,
  NavLink,
  Stack,
  SupposeButton,
  Toasts,
  useWidth,
} from '../ui'
import { icons } from './kinds'
import DraftBar from './DraftBar.vue'
import Search from './Search.vue'
import SupposedBar from './SupposedBar.vue'
import { useThemeSwitch } from './theme'

// Around every page: the bar with the lists, the search, Suppose, the findings, the
// theme, whether the page hears the server and the menu with the view, the draft being
// written and the hypotheses supposed below it, an export's notice across the window
// under the bar, and the notices at the corner. Before the passphrase is given, the bar
// holds the name and the theme alone.
const {
  session,
  index,
  findings,
  live,
  signIn,
  view,
  supposing,
  supposed,
  drafting,
  toasts,
  writes,
  setView,
  setSupposing,
  dismiss,
  act,
  undoing,
} = useCorpus()
const client = useClient()
const route = useRoute()
const router = useRouter()
const width = useWidth()
const { theme, toggle } = useThemeSwitch()
const fits = ref(true)
const drawer = ref(false)

watch(() => route.fullPath, () => (drawer.value = false))
watchEffect(() => (document.title = session.value?.project || app.title))

const lists = computed(() =>
  kindsInOrder.map((kind) => ({
    kind,
    to: `/${listed[kind]}`,
    icon: icons[kind],
    label: kinds[kind].many,
    count: index.value?.[listed[kind]].length,
  })),
)
const found = computed(() =>
  findings.value ? findings.value.errors + findings.value.warnings + findings.value.todo.length : undefined,
)
const locked = computed(() => session.value?.view != null)
const supposable = computed(() => writes())
const shown = computed(() => (locked.value ? session.value!.view! : view.value))
const themeIcon = computed(() => (theme.value === 'dark' ? 'sun' : 'moon'))
// an export's pages are read with what it was made with, which Info says
const supposes = computed(() => !!session.value?.live && (supposed.value.length > 0 || session.value.with.length > 0))
const notice = computed(() => session.value?.export?.notice)

function hide(which: 'private' | 'living', on: boolean) {
  setView({ ...view.value, [which]: on })
}
</script>

<template>
  <Bar v-model:fits="fits">
    <template v-if="session" #toggle>
      <Drawer v-model:open="drawer" :title="frame.lists" side="left">
        <template #trigger><IconButton icon="menu" :label="frame.lists" /></template>
        <Stack gap="1">
          <NavLink to="/" icon="family" drawer>{{ frame.home }}</NavLink>
          <NavLink v-for="list in lists" :key="list.kind" :to="list.to" :icon="list.icon" :count="list.count" drawer>
            {{ list.label }}
          </NavLink>
        </Stack>
      </Drawer>
    </template>
    <template #brand>
      <Brand :title="app.title" to="/" />
    </template>
    <template v-if="session" #nav>
      <NavLink v-for="list in lists" :key="list.kind" :to="list.to">{{ list.label }}</NavLink>
    </template>
    <Search v-if="session" :index="index" />
    <template #end>
      <template v-if="session">
        <Live v-if="session.live && width === 'wide'" :up="live">{{ live ? frame.live : frame.offline }}</Live>
        <SupposeButton
          v-if="session.live && width !== 'phone'"
          :label="frame.suppose"
          :pressed="drafting"
          :disabled="!supposable"
          @click="setSupposing(!supposing)"
        />
        <IconButton icon="flag" :label="frame.findings" :badge="found" to="/findings" />
        <IconButton v-if="width === 'wide'" :icon="themeIcon" :label="frame.theme" @click="toggle" />
        <Menu align="end">
          <template #trigger><IconButton icon="more" :label="frame.more" /></template>
          <template v-if="width !== 'wide'">
            <MenuItem :icon="themeIcon" @select="toggle">{{ frame.theme }}</MenuItem>
            <MenuItem v-if="session.live" disabled><Live :up="live">{{ live ? frame.live : frame.offline }}</Live></MenuItem>
            <MenuItem
              v-if="session.live && width === 'phone'"
              icon="flask"
              :checked="drafting"
              :disabled="!supposable"
              @update:checked="(on) => setSupposing(on === true)"
            >
              {{ frame.suppose }}
            </MenuItem>
            <MenuSeparator />
          </template>
          <template v-if="session.live">
            <MenuItem :checked="shown.private" :disabled="locked" @update:checked="(on) => hide('private', on === true)">
              {{ frame.hidePrivate }}
            </MenuItem>
            <MenuItem :checked="shown.living" :disabled="locked" @update:checked="(on) => hide('living', on === true)">
              {{ frame.hideLiving }}
            </MenuItem>
            <MenuSeparator />
            <MenuItem icon="gear" @select="router.push('/settings')">{{ frame.settings }}</MenuItem>
          </template>
          <MenuItem v-else icon="archive" @select="router.push('/info')">{{ frame.info }}</MenuItem>
        </Menu>
      </template>
      <IconButton v-else-if="signIn" :icon="themeIcon" :label="frame.theme" @click="toggle" />
    </template>
    <template v-if="session && (drafting || supposes)" #below>
      <DraftBar v-if="drafting" />
      <SupposedBar v-if="supposes" />
    </template>
  </Bar>
  <Banner v-if="notice" data-notice>{{ notice }}</Banner>
  <slot />
  <Toasts :toasts="toasts" @dismiss="(key) => dismiss(Number(key))" @act="(key) => act(Number(key))" />
  <ChangeDialog
    v-model:open="undoing"
    :title="said.undo"
    :confirm="said.undo"
    :make="(dry, sources) => client.undo({ dry, sources })"
    path="undo"
  />
</template>
