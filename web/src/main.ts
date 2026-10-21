import { createApp } from 'vue'
import './ui/tokens.css'
import './ui/base.css'
import App from './App.vue'
import router from './router'
import { Client } from './api/client'
import { corpusKey, createCorpus } from './api/corpus'
import { clientKey } from './api/use'
import { applyKeptTheme } from './frame/theme'

applyKeptTheme()
const client = new Client()
createApp(App).provide(clientKey, client).provide(corpusKey, createCorpus(client)).use(router).mount('#app')
