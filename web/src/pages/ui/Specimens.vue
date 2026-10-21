<script setup lang="ts">
import { ref } from 'vue'
import {
  Badge,
  Banner,
  Bar,
  Block,
  Brand,
  Button,
  Card,
  Chart,
  Chip,
  Code,
  Columns,
  Combobox,
  DateInput,
  Definition,
  Definitions,
  Dialog,
  Drawer,
  Entry,
  Field,
  Fieldset,
  Grid,
  Hero,
  Icon,
  IconButton,
  Lightbox,
  Live,
  Markdown,
  Menu,
  MenuItem,
  MenuSeparator,
  Moment,
  Name,
  NavLink,
  Notice,
  Page,
  Palette,
  PersonCard,
  Portrait,
  Prose,
  Popover,
  Regions,
  SearchButton,
  Segmented,
  Select,
  Stack,
  Stat,
  SupposeButton,
  Switch,
  Table,
  Tabs,
  Tag,
  Text,
  TextArea,
  Theme,
  Scan,
  Thumb,
  Timeline,
  Toasts,
  Tree,
  Value,
  type Sort,
} from '../../ui'
import { charts, columns, depicted, untagged, found, iconNames, people, person, picture, places, regions, sexes, shapes, tabs, tags, words, yaml } from '../../ui/gallery/specimens'

// Every component in every state it has, for one theme; /ui draws it twice.
const props = defineProps<{ theme: 'light' | 'dark' }>()

const given = ref<string>(words.givenValue)
const empty = ref('')
const slug = ref<string>(words.slugValue)
const body = ref<string>(words.bodyValue)
const date = ref<string>(words.dateValue)
const bad = ref<string>(words.dateBad)
const sex = ref<string>()
const chosen = ref('M')
const living = ref(true)
const off = ref(false)
const shape = ref('about')
const popover = ref(false)
const dialog = ref(false)
const drawer = ref(false)
const wide = ref(false)
const lightbox = ref(false)
const drawn = ref<number[]>()
const drawings = ref<Awaited<ReturnType<typeof charts>>>()
void charts().then((c) => (drawings.value = c))
const nav = ref(false)
const shown = ref(true)
const tab = ref('life')
const sort = ref<Sort | undefined>({ key: 'name' })
const opened = ref<string>()
const pressed = ref(true)
const fits = ref(true)
const supposing = ref(false)
const palette = ref(false)
const query = ref<string>(words.query)
const place = ref<string>(words.placeValue)
const surname = ref<string>(words.surnameValue)
const placed = places.map((p) => ({ value: p.toLowerCase().replace(/ /g, '-'), label: p }))
const toasts = ref<{ key: number; text: string }[]>([])
let toasted = 0
const toast = () => toasts.value.push({ key: ++toasted, text: words.toast })
</script>

<template>
  <Page :title="`${words.title} · ${theme === 'light' ? words.light : words.dark}`">
    <Stack gap="10">
      <Block :title="words.frame">
        <Stack gap="3">
          <Bar v-model:fits="fits" still>
            <template #toggle><IconButton icon="menu" :label="words.nav" /></template>
            <template #brand><Brand :title="words.origins" to="/ui" /></template>
            <template #nav>
              <NavLink to="/ui">{{ words.people }}</NavLink>
              <NavLink to="/ui#families">{{ words.families }}</NavLink>
              <NavLink to="/ui#places">{{ words.places }}</NavLink>
              <NavLink to="/ui#sources">{{ words.sources }}</NavLink>
            </template>
            <SearchButton :label="words.search" :keys="words.searchKeys" @click="palette = true" />
            <template #end>
              <SupposeButton :label="words.suppose" :pressed="supposing" @click="supposing = !supposing" />
              <IconButton icon="flag" :label="words.findings" :badge="4" to="/ui" />
              <Live up>{{ words.live }}</Live>
            </template>
          </Bar>
          <Stack row wrap gap="4" align="center">
            <Brand :title="words.origins" />
            <Live :up="false">{{ words.offline }}</Live>
            <SupposeButton :label="words.suppose" disabled />
            <Button @click="toast">{{ words.notify }}</Button>
          </Stack>
          <Stack gap="1">
            <NavLink to="/ui" icon="person" :count="2000" drawer>{{ words.people }}</NavLink>
            <NavLink to="/ui#places" icon="place" :count="6" drawer>{{ words.places }}</NavLink>
          </Stack>
          <Palette v-model:open="palette" v-model:query="query" :label="words.search" :groups="found" :keys="words.keys" />
          <Toasts :toasts="toasts" @dismiss="(key) => (toasts = toasts.filter((t) => t.key !== key))" />
        </Stack>
      </Block>

      <Block :title="words.buttons">
        <Stack gap="3">
          <Stack row wrap gap="2">
            <Button>{{ words.cancel }}</Button>
            <Button variant="primary">{{ words.save }}</Button>
            <Button variant="ghost" icon="git">{{ words.history }}</Button>
            <Button variant="danger">{{ words.remove }}</Button>
            <Button variant="danger" solid>{{ words.remove }}</Button>
            <Button variant="add" icon="plus">{{ words.addEvent }}</Button>
            <Button href="#">{{ words.link }}</Button>
          </Stack>
          <Stack row wrap gap="2">
            <Button disabled>{{ words.cancel }}</Button>
            <Button variant="primary" disabled>{{ words.save }}</Button>
            <Button size="sm">{{ words.cancel }}</Button>
            <Button size="sm" variant="primary">{{ words.save }}</Button>
            <IconButton icon="more" :label="words.more" />
            <IconButton icon="git" :label="words.history" bordered />
            <IconButton icon="eye-off" :label="words.show" pressed bordered />
            <IconButton icon="flask" :label="words.supposing" :badge="3" />
            <IconButton icon="undo" :label="words.undo" disabled />
          </Stack>
          <Theme :theme="props.theme" supposing>
            <Stack row wrap gap="2">
              <Text caps>{{ words.supposing }}</Text>
              <Button variant="primary">{{ words.save }}</Button>
              <Button variant="add" icon="plus">{{ words.addEvent }}</Button>
              <Switch v-model="living" :label="words.living" />
            </Stack>
          </Theme>
        </Stack>
      </Block>

      <Block :title="words.fields">
        <Grid :cols="2" gap="5">
          <Field v-model="given" :label="words.given" path="name.given" />
          <Field v-model="empty" :label="words.surname" :placeholder="words.surname" path="name.surnames[0]" />
          <Field v-model="empty" :label="words.page" :hint="words.pageHint" />
          <Field v-model="slug" :label="words.slug" :error="words.slugError" mono />
          <Field v-model="given" :label="words.given" disabled />
          <Select v-model="sex" :label="words.sex" :options="sexes" :placeholder="words.choose" path="sex" />
          <Select v-model="chosen" :label="words.sex" :options="sexes" />
          <Select v-model="chosen" :label="words.sex" :options="sexes" disabled />
          <DateInput v-model="date" :label="words.date" :reads="words.dateReads" path="events[0].date" />
          <DateInput v-model="bad" :label="words.date" :error="words.dateError" />
          <DateInput v-model="date" :label="words.date" disabled />
          <Segmented v-model="shape" :label="words.shape" :options="shapes" />
          <Segmented v-model="shape" :label="words.shape" :options="shapes" disabled />
          <Stack gap="2">
            <Switch v-model="living" :label="words.living" />
            <Switch v-model="off" :label="words.private" />
            <Switch v-model="living" :label="words.living" disabled />
          </Stack>
        </Grid>
        <Grid :cols="2" gap="5">
          <TextArea v-model="body" :label="words.body" path="body" prose />
          <TextArea v-model="empty" :label="words.transcription" :placeholder="words.transcription" :rows="3" />
          <TextArea v-model="body" :label="words.body" :error="words.dateError" :rows="3" />
          <TextArea v-model="body" :label="words.body" :rows="3" disabled />
          <Markdown v-model="body" :html="person.prose" :label="words.body" path="body" />
          <Markdown v-model="body" :html="person.prose" :label="words.body" :rows="3" disabled />
        </Grid>
      </Block>

      <Block :title="words.values">
        <Grid :cols="2" gap="5">
          <Definitions>
            <Definition :term="words.sex" path="sex"><Value path="sex" :label="words.sex">M</Value></Definition>
            <Definition :term="words.date" path="events[0].date">
              <Value path="events[0].date" :label="words.date" mono>{{ words.dateValue }}</Value>
            </Definition>
            <Definition :term="words.occupation" path="occupation">
              <Value path="occupation" :label="words.occupation" disabled>{{ words.occupationValue }}</Value>
            </Definition>
            <Definition :term="words.place" path="events[0].place">
              <Value path="events[0].place" :label="words.place" marked>{{ words.placeValue }}</Value>
            </Definition>
            <Definition :term="words.place">
              <Button size="sm" variant="add" icon="plus">{{ words.place }}</Button>
            </Definition>
          </Definitions>
          <Definitions compact>
            <Definition :term="words.sex"><Value>M</Value></Definition>
            <Definition :term="words.place"><Value>{{ words.placeValue }}</Value></Definition>
          </Definitions>
          <Combobox v-model="place" :label="words.place" :options="placed" path="events[0].place" />
          <Combobox v-model="empty" :label="words.place" :options="placed" :placeholder="words.place" :error="words.dateError" />
          <Fieldset :legend="words.names" path="names">
            <Entry path="names[0]">
              <Field v-model="given" :label="words.given" />
              <Field v-model="surname" :label="words.surname" />
              <template #actions><IconButton icon="x" :label="words.takeOut" /></template>
            </Entry>
            <template #add><Button size="sm" variant="add" icon="plus">{{ words.add }}</Button></template>
          </Fieldset>
          <Fieldset :legend="words.names">
            <Entry><Field v-model="given" :label="words.given" disabled /></Entry>
          </Fieldset>
        </Grid>
      </Block>

      <Block :title="words.overlays">
        <Stack row wrap gap="2">
          <Popover v-model:open="popover" :path="words.popoverPath" :label="words.date">
            <template #trigger><Button>{{ words.popover }}</Button></template>
            <DateInput v-model="date" :label="words.date" :reads="words.dateReads" />
            <template #actions>
              <Button size="sm" @click="popover = false">{{ words.cancel }}</Button>
              <Button size="sm" variant="primary" @click="popover = false">{{ words.save }}</Button>
            </template>
          </Popover>
          <Dialog v-model:open="dialog" :title="words.dialogTitle">
            <template #trigger><Button variant="danger">{{ words.dialog }}</Button></template>
            <Text as="p" tone="muted">{{ words.dialogBody }}</Text>
            <template #actions>
              <Button @click="dialog = false">{{ words.cancel }}</Button>
              <Button variant="danger" solid @click="dialog = false">{{ words.remove }}</Button>
            </template>
          </Dialog>
          <Button icon="note" @click="drawer = true">{{ words.drawer }}</Button>
          <Drawer v-model:open="drawer" :title="words.drawerTitle" :path="words.drawerPath">
            <Stack gap="5">
              <DateInput v-model="date" :label="words.date" :reads="words.dateReads" />
              <Field v-model="given" :label="words.given" />
              <TextArea v-model="body" :label="words.body" prose />
            </Stack>
            <template #actions>
              <Button @click="drawer = false">{{ words.cancel }}</Button>
              <Button variant="primary" @click="drawer = false">{{ words.save }}</Button>
            </template>
          </Drawer>
          <Button icon="note" @click="wide = true">{{ words.body }}</Button>
          <Drawer v-model:open="wide" :title="words.body" path="body" wide>
            <Markdown v-model="body" :html="person.prose" :label="words.body" path="body" />
            <template #actions>
              <Button @click="wide = false">{{ words.cancel }}</Button>
              <Button variant="primary" @click="wide = false">{{ words.save }}</Button>
            </template>
          </Drawer>
          <Button icon="menu" @click="nav = true">{{ words.nav }}</Button>
          <Drawer v-model:open="nav" :title="words.nav" side="left">
            <Stack gap="2">
              <Chip icon="person">{{ words.tableLabel }}</Chip>
              <Chip icon="place">{{ words.source }}</Chip>
            </Stack>
          </Drawer>
          <Menu>
            <template #trigger><IconButton icon="more" :label="words.more" bordered /></template>
            <MenuItem icon="note">{{ words.edit }}</MenuItem>
            <MenuItem icon="tag" :hint="words.hint">{{ words.rename }}</MenuItem>
            <MenuItem icon="users">{{ words.merge }}</MenuItem>
            <MenuItem v-model:checked="shown">{{ words.show }}</MenuItem>
            <MenuItem disabled>{{ words.undo }}</MenuItem>
            <MenuSeparator />
            <MenuItem icon="x" danger>{{ words.remove }}</MenuItem>
          </Menu>
        </Stack>
      </Block>

      <Block :title="words.surfaces">
        <Grid fit="card">
          <Card :title="words.details">
            <Text as="p">{{ words.cardBody }}</Text>
          </Card>
          <Card :title="words.research" tone="research">
            <template #action><IconButton icon="plus" :label="words.addEvent" /></template>
            <Text as="p">{{ words.researchBody }}</Text>
          </Card>
          <Card :title="words.warn" tone="warn">
            <Text as="p">{{ words.warnBody }}</Text>
            <template #footer><Button size="sm">{{ words.edit }}</Button></template>
          </Card>
          <Card flush>
            <Notice tone="suppose" icon="flask">{{ words.draft }}</Notice>
          </Card>
        </Grid>
        <Grid fit="card" packed>
          <Card :title="words.details"><Text as="p">{{ words.cardBody }}</Text></Card>
          <Card :title="words.research" tone="research">
            <Stack gap="2">
              <Text as="p">{{ words.researchBody }}</Text>
              <Text as="p">{{ words.cardBody }}</Text>
              <Text as="p">{{ words.warnBody }}</Text>
            </Stack>
          </Card>
          <Card :title="words.warn" tone="warn"><Text as="p">{{ words.warnBody }}</Text></Card>
        </Grid>
        <Tabs v-model="tab" :tabs="tabs" :label="words.sections">
          <template #life><Text as="p">{{ words.cardBody }}</Text></template>
          <template #family><Text as="p" tone="muted">{{ words.researchBody }}</Text></template>
          <template #sources><Chip tone="source" icon="book">{{ words.source }}</Chip></template>
        </Tabs>
      </Block>

      <Block :title="words.table">
        <template #actions><Badge mono>{{ people.length }}</Badge></template>
        <Table
          v-model:sort="sort"
          :columns="columns"
          :rows="people"
          :row-key="(p) => p.slug"
          :label="words.tableLabel"
          :current="opened"
          box
          data-testid="table"
          @open="(p) => (opened = p.slug)"
        >
          <template #cell-name="{ row }">
            <Text face="serif" size="lg">{{ row.name }}</Text>
          </template>
        </Table>
        <Table :columns="columns" :rows="[]" :row-key="(p) => p.slug" :label="words.tableLabel">
          <template #empty>{{ words.empty }}</template>
        </Table>
      </Block>

      <Block :title="words.overview">
        <Stack gap="4">
          <Grid fit="stat" gap="3">
            <Stat :count="people.length" :label="words.people" icon="person" to="/ui" />
            <Stat :count="places.length" :label="words.places" icon="place" to="/ui" />
            <Stat :count="0" :label="words.sources" icon="book" to="/ui" />
          </Grid>
          <Card>
            <Tree :items="regions" :under="(r) => r.under" :item-key="(r) => r.slug" :label="words.tree">
              <template #default="{ item, depth }">
                <Stack row gap="2" align="baseline">
                  <Chip to="/ui">{{ item.name }}</Chip>
                  <Text tone="faint" size="sm">{{ depth }}</Text>
                </Stack>
              </template>
            </Tree>
          </Card>
          <Code :label="words.file">{{ yaml }}</Code>
        </Stack>
      </Block>

      <Block :title="words.marks">
        <Stack gap="3">
          <Stack row wrap gap="2">
            <Tag v-for="tag in tags" :key="tag.name" :name="tag.name" :color="tag.color" />
          </Stack>
          <Stack row wrap gap="2">
            <Chip>{{ words.otherName }}</Chip>
            <Chip :kind="words.nickname">{{ words.otherName }}</Chip>
            <Chip icon="tag" :pressed="pressed" @click="pressed = !pressed">{{ words.filter }}</Chip>
            <Chip removable>{{ words.filter }}</Chip>
            <Chip add>{{ words.add }}</Chip>
            <Chip tone="source" icon="book">{{ words.source }}</Chip>
            <Chip tone="suppose" icon="flask">{{ words.supposed }}</Chip>
            <Chip to="/ui">{{ words.link }}</Chip>
          </Stack>
          <Stack row wrap gap="2">
            <Badge>{{ words.open }}</Badge>
            <Badge tone="good">{{ words.living }}</Badge>
            <Badge tone="warn">{{ words.open }}</Badge>
            <Badge tone="danger">{{ words.private }}</Badge>
            <Badge tone="suppose">{{ words.supposing }}</Badge>
            <Badge tone="accent">{{ words.open }}</Badge>
            <Badge tone="faint" struck>{{ words.rejected }}</Badge>
            <Badge mono>B2</Badge>
          </Stack>
        </Stack>
      </Block>

      <Block :title="person.heading">
        <Stack gap="6">
          <Hero>
            <template #portrait>
              <Portrait size="lg" :src="picture" :region="person.region" :initials="person.initials" sex="M" />
            </template>
            <template #title><Name :parts="person.parts" /></template>
            <Text italic tone="muted">{{ person.written }}</Text>
            <Text tone="muted">{{ person.lifespan }}</Text>
            <template #actions><IconButton icon="more" :label="words.actions" /></template>
          </Hero>
          <Stack row wrap gap="3" align="center">
            <Icon v-for="name in iconNames" :key="name" :name="name" />
          </Stack>
          <Stack row wrap gap="3" align="center">
            <Portrait :initials="person.initials" sex="M" />
            <Portrait :initials="person.initials" sex="F" />
            <Portrait :initials="person.initials" />
            <Portrait as="button" :label="words.edit" :src="picture" :initials="person.initials" />
            <Portrait as="button" :label="words.edit" :src="picture" :initials="person.initials" marked />
            <Thumb :src="picture" :title="person.thumb" />
            <Thumb as="button" :title="person.thumb" />
            <Thumb :src="picture" :title="person.thumb" :regions="depicted" />
            <Lightbox v-model:open="lightbox" v-model:drawn="drawn" :title="person.thumb" :src="picture" :regions="depicted" draw>
              <template #trigger><Thumb as="button" :src="picture" :title="person.thumb" :regions="depicted" /></template>
              <Text tone="muted">{{ person.lifespan }}</Text>
            </Lightbox>
            <Thumb size="sm" :src="picture" :title="person.thumb" />
            <Scan :src="picture" :region="person.region" :title="person.thumb" />
            <Scan :title="person.thumb" />
            <Scan :src="picture" :title="person.thumb"><Regions :regions="depicted" :drawing="untagged" labelled /></Scan>
          </Stack>
          <Grid fit="card" gap="2">
            <PersonCard
              :name="person.name"
              to="/ui"
              :lifespan="person.years"
              sex="M"
              :src="picture"
              :region="person.region"
              :initials="person.initials"
            />
            <PersonCard :name="person.partner" to="/ui" :lifespan="person.partnerYears" sex="F" initials="MB" />
            <PersonCard :name="person.child" to="/ui" :relation="person.adopted" sex="F" initials="AR" />
            <PersonCard :name="person.disproven" to="/ui" struck initials="AR" />
            <PersonCard :name="person.unnamed" ghost initials="R" />
          </Grid>
          <Timeline>
            <Moment v-for="(m, i) in person.moments" :key="i" :year="m.year" :quiet="'quiet' in m" :span="'span' in m">
              <Text :tone="'quiet' in m ? 'muted' : undefined">{{ m.text }}</Text>
            </Moment>
          </Timeline>
          <Prose :html="person.prose" />
          <template v-if="drawings">
            <Chart
              :boxes="drawings.chart.boxes"
              :lines="drawings.chart.lines"
              :width="drawings.chart.width"
              :height="drawings.chart.height"
              :box="drawings.chart.box"
              :label="person.name"
            />
            <Chart
              :boxes="drawings.gapped.boxes"
              :lines="drawings.gapped.lines"
              :width="drawings.gapped.width"
              :height="drawings.gapped.height"
              :box="drawings.gapped.box"
              :label="person.child"
            />
          </template>
        </Stack>
      </Block>

      <Block :title="words.notices">
        <Stack gap="2">
          <Banner>{{ words.banner }}</Banner>
          <Notice>{{ words.info }}</Notice>
          <Notice tone="warn">{{ words.warning }}</Notice>
          <Notice tone="danger">{{ words.danger }}</Notice>
          <Notice tone="good" icon="check">{{ words.good }}</Notice>
          <Notice tone="suppose" icon="flask">
            {{ words.draft }}
            <template #actions><Button size="sm" variant="primary">{{ words.keep }}</Button></template>
          </Notice>
          <Notice tone="ink" dismissible>
            {{ words.toast }}
            <template #actions><Button size="sm" variant="ghost" icon="undo">{{ words.undo }}</Button></template>
          </Notice>
        </Stack>
      </Block>

      <Block :title="words.text">
        <Stack gap="1">
          <Text caps>{{ words.details }}</Text>
          <Text face="serif" size="xl">{{ words.serif }}</Text>
          <Text tone="muted">{{ words.muted }}</Text>
          <Text tone="faint" size="sm">{{ words.faint }}</Text>
          <Text face="mono">{{ words.mono }}</Text>
          <Text italic tone="act">{{ words.supposed }}</Text>
        </Stack>
      </Block>

      <Block :title="words.layout">
        <Columns>
          <Grid :cols="3" gap="2">
            <Card v-for="n in 6" :key="n">{{ words.cell }}</Card>
          </Grid>
          <template #side>
            <Card :title="words.side"><Text tone="muted">{{ words.cardBody }}</Text></Card>
          </template>
        </Columns>
      </Block>
    </Stack>
  </Page>
</template>
