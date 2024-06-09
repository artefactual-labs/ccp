import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createConnectTransport } from '@connectrpc/connect-web'

import App from './App.vue'
import router from './router'
import { transportKey, adminClientKey } from './client'
import { createPromiseClient } from '@connectrpc/connect'
import { AdminService } from './gen/archivematica/ccp/admin/v1beta1/admin_connect'

const app = createApp(App)

app.use(createPinia())
app.use(router)

// Provide transport.
const loc = window.location
const baseUrl = `${loc.protocol}//${loc.hostname}:${loc.port}/api`
const transport = createConnectTransport({ baseUrl })
app.provide(transportKey, transport)

// Provide Admin Service client.
const client = createPromiseClient(AdminService, transport)
app.provide(adminClientKey, client)

app.mount('#app')
