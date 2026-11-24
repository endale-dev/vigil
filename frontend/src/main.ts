import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { useSession } from './composables/useSession'
import { router } from './router'

void useSession().initialize().catch(() => null)

createApp(App).use(router).mount('#app')
