<template>
  <div>
    <div class="page-head">
      <div>
        <h1>中间件流水线</h1>
        <p>编译期注册，运行期按配置启停。JWT / 限流 / 日志 / CORS / 改写 / 头注入 / IP 过滤。</p>
      </div>
      <button class="btn" @click="issueToken">签发演示 JWT</button>
    </div>
    <div class="mw-grid">
      <div class="card" v-for="m in rows" :key="m.name">
        <div class="kicker">{{ m.scope }}</div>
        <h2 style="margin:8px 0 4px">{{ m.name }}</h2>
        <p style="color:var(--fog);min-height:40px">{{ m.description }}</p>
        <div class="field">
          <label>启用</label>
          <select v-model="m.enabled">
            <option :value="true">是</option>
            <option :value="false">否</option>
          </select>
        </div>
        <div class="field">
          <label>配置 JSON <span class="req">*</span></label>
          <textarea v-model="m.cfgText" />
          <div v-if="m.err" class="err">{{ m.err }}</div>
        </div>
        <button class="btn" @click="save(m)">保存并热更新</button>
      </div>
    </div>
    <p v-if="token" class="card" style="margin-top:16px">
      演示 Token（1 小时）<br />
      <code class="mono" style="word-break:break-all">{{ token }}</code>
    </p>
  </div>
</template>

<script setup>
import { inject, onMounted, ref } from 'vue'
import { api } from '../api'

const toast = inject('toast')
const rows = ref([])
const token = ref('')

async function load() {
  const list = await api.middlewares() || []
  rows.value = list.map((m) => ({
    ...m,
    cfgText: JSON.stringify(m.config || {}, null, 2),
    err: '',
  }))
}
function validate(m) {
  if (!m.cfgText?.trim()) {
    m.err = '配置不能为空'
    return null
  }
  try {
    const cfg = JSON.parse(m.cfgText)
    m.err = ''
    return cfg
  } catch {
    m.err = 'JSON 无法解析'
    return null
  }
}
async function save(m) {
  const cfg = validate(m)
  if (!cfg) { toast.push('请修正配置 JSON', 'error'); return }
  try {
    await api.updateMW(m.name, { name: m.name, enabled: m.enabled, config: cfg })
    toast.push(m.name + ' 已热更新')
    await load()
  } catch (err) { toast.push(err.message, 'error') }
}
async function issueToken() {
  try {
    const d = await api.demoToken()
    token.value = d.token
    toast.push('已签发，过期 ' + d.expires_at)
  } catch (err) { toast.push(err.message, 'error') }
}
onMounted(load)
</script>
