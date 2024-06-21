import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import { client } from './client'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(client)

app.mount('#app')
