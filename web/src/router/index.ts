import { createRouter, createWebHashHistory, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { exported, kindsInOrder } from '../api/client'
import { listed } from '../search'

// The component gallery is a development page: the build drops the branch, and with
// it the page and its specimens.
const gallery: RouteRecordRaw[] = import.meta.env.DEV
  ? [{ path: '/ui', name: 'ui', component: () => import('../pages/UiPage.vue') }]
  : []

const lists: RouteRecordRaw[] = kindsInOrder.map((kind) => ({
  path: `/${listed[kind]}`,
  name: listed[kind],
  component: () => import('../pages/ListPage.vue'),
  props: { kind },
}))

// A record is at its ref, kind and slug, as the server writes a link to one.
const records: RouteRecordRaw[] = kindsInOrder.map((kind) => ({
  path: `/${kind}/:slug`,
  name: kind,
  component: () => import('../pages/RecordPage.vue'),
  props: (route) => ({ kind, slug: route.params.slug }),
}))

// An export's page stays where it was opened, a folder having no route but its files,
// and takes the route after the hash; the server answers every route with the page.
const router = createRouter({
  history: exported ? createWebHashHistory() : createWebHistory('/'),
  routes: [
    { path: '/', name: 'home', component: () => import('../pages/HomePage.vue') },
    ...lists,
    ...records,
    { path: '/findings', name: 'findings', component: () => import('../pages/FindingsPage.vue') },
    { path: '/settings', name: 'settings', component: () => import('../pages/SettingsPage.vue') },
    { path: '/info', name: 'info', component: () => import('../pages/SettingsPage.vue') },
    ...gallery,
    { path: '/:rest(.*)*', redirect: '/' },
  ],
})

export default router
