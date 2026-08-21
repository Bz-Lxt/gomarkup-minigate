import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'
import Dashboard from './views/Dashboard.vue'
import Routes from './views/Routes.vue'
import Upstreams from './views/Upstreams.vue'
import Middlewares from './views/Middlewares.vue'
import Config from './views/Config.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Dashboard },
    { path: '/routes', component: Routes },
    { path: '/upstreams', component: Upstreams },
    { path: '/middlewares', component: Middlewares },
    { path: '/config', component: Config },
  ],
})

createApp(App).use(router).mount('#app')
