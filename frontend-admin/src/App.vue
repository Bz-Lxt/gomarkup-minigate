<template>
  <div class="shell">
    <aside class="side">
      <div class="brand">MiniGate <small>NIGHT BRIDGE</small></div>
      <nav class="nav">
        <router-link to="/">仪表盘</router-link>
        <router-link to="/routes">路由</router-link>
        <router-link to="/upstreams">上游</router-link>
        <router-link to="/middlewares">中间件</router-link>
        <router-link to="/config">配置</router-link>
      </nav>
      <div class="side-foot mono">GMT+8 · 热更新无需重启</div>
    </aside>
    <main class="main">
      <router-view />
    </main>
    <div class="toasts">
      <div v-for="t in toasts" :key="t.id" class="toast" :class="{ error: t.type === 'error' }">
        <span>{{ t.text }}</span>
        <button class="btn ghost" style="padding:2px 8px" @click="remove(t.id)">×</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { provide, ref } from 'vue'

const toasts = ref([])
let seq = 0
function push(text, type = 'ok') {
  const id = ++seq
  toasts.value.push({ id, text, type })
  setTimeout(() => remove(id), 5000)
}
function remove(id) {
  toasts.value = toasts.value.filter((t) => t.id !== id)
}
provide('toast', { push, remove })
</script>
