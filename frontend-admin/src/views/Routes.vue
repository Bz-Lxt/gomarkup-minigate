<template>
  <div>
    <div class="page-head">
      <div>
        <h1>路由表</h1>
        <p>Radix Tree 匹配规则。保存后立即热更新，无需重启网关。</p>
      </div>
      <button class="btn" @click="openCreate">新建路由</button>
    </div>
    <div class="card table-wrap">
      <table>
        <thead>
          <tr>
            <th>状态</th><th>名称</th><th>路径</th><th>方法</th><th>上游</th><th>中间件</th><th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in rows" :key="r.id">
            <td><span class="led" :class="r.enabled ? 'on' : 'off'"></span>{{ r.enabled ? '启用' : '停用' }}</td>
            <td>{{ r.name }}</td>
            <td class="mono">{{ r.path }}</td>
            <td class="mono">{{ (r.methods || []).join(', ') }}</td>
            <td>{{ r.upstream_id }}</td>
            <td>{{ (r.middlewares || []).join(', ') || '—' }}</td>
            <td class="row-actions">
              <button class="btn ghost" @click="edit(r)">编辑</button>
              <button class="btn ghost" @click="toggle(r)">{{ r.enabled ? '禁用' : '启用' }}</button>
              <button class="btn danger" @click="askDel(r)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="drawer" class="drawer-mask" @click.self="drawer = false">
      <div class="drawer">
        <h2>{{ form.id && !creating ? '编辑路由' : '新建路由' }}</h2>
        <div class="field">
          <label>ID <span class="req">*</span></label>
          <input v-model="form.id" :disabled="!creating" />
          <div v-if="errors.id" class="err">{{ errors.id }}</div>
        </div>
        <div class="field">
          <label>名称 <span class="req">*</span></label>
          <input v-model="form.name" />
          <div v-if="errors.name" class="err">{{ errors.name }}</div>
        </div>
        <div class="field">
          <label>路径 <span class="req">*</span></label>
          <input v-model="form.path" placeholder="/api/{id} 或 /echo/*" />
          <div v-if="errors.path" class="err">{{ errors.path }}</div>
        </div>
        <div class="field">
          <label>方法（逗号分隔）</label>
          <input v-model="methodsText" />
        </div>
        <div class="field">
          <label>Host（可空）</label>
          <input v-model="form.host" />
        </div>
        <div class="field">
          <label>上游 <span class="req">*</span></label>
          <select v-model="form.upstream_id">
            <option value="">请选择</option>
            <option v-for="u in ups" :key="u.id" :value="u.id">{{ u.name }} ({{ u.id }})</option>
          </select>
          <div v-if="errors.upstream_id" class="err">{{ errors.upstream_id }}</div>
        </div>
        <div class="field">
          <label>Strip Prefix</label>
          <input v-model="form.strip_prefix" />
        </div>
        <div class="field">
          <label>中间件（逗号分隔 jwt,ratelimit,logger）</label>
          <input v-model="mwText" />
        </div>
        <div class="field">
          <label>优先级</label>
          <input v-model.number="form.priority" type="number" min="0" />
        </div>
        <div class="field">
          <label>启用</label>
          <select v-model="form.enabled">
            <option :value="true">是</option>
            <option :value="false">否</option>
          </select>
        </div>
        <button class="btn" @click="save">保存</button>
        <button class="btn ghost" style="margin-left:8px" @click="drawer=false">取消</button>
      </div>
    </div>

    <div v-if="delTarget" class="modal-mask">
      <div class="modal">
        <h3>删除路由 {{ delTarget.name }}？</h3>
        <p>此操作会立即写入 YAML 并热更新。</p>
        <button class="btn danger" @click="confirmDel">确认删除</button>
        <button class="btn ghost" style="margin-left:8px" @click="delTarget=null">取消</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject, onMounted, ref } from 'vue'
import { api } from '../api'

const toast = inject('toast')
const rows = ref([])
const ups = ref([])
const drawer = ref(false)
const creating = ref(true)
const form = ref({})
const methodsText = ref('GET')
const mwText = ref('')
const errors = ref({})
const delTarget = ref(null)

function blank() {
  return { id: '', name: '', path: '', host: '', upstream_id: '', strip_prefix: '', enabled: true, priority: 10, methods: ['GET'], middlewares: [] }
}

async function load() {
  rows.value = await api.routes() || []
  ups.value = await api.upstreams() || []
}
function openCreate() {
  creating.value = true
  form.value = blank()
  methodsText.value = 'GET'
  mwText.value = ''
  errors.value = {}
  drawer.value = true
}
function edit(r) {
  creating.value = false
  form.value = { ...r }
  methodsText.value = (r.methods || []).join(',')
  mwText.value = (r.middlewares || []).join(',')
  errors.value = {}
  drawer.value = true
}
function validate() {
  const e = {}
  if (!form.value.id?.trim()) e.id = 'ID 必填，建议小写短横线'
  if (!form.value.name?.trim()) e.name = '名称必填'
  if (!form.value.path?.startsWith('/')) e.path = '路径必须以 / 开头'
  if (!form.value.upstream_id) e.upstream_id = '必须选择上游'
  errors.value = e
  return Object.keys(e).length === 0
}
async function save() {
  if (!validate()) {
    toast.push('请修正表单错误', 'error')
    return
  }
  const body = {
    ...form.value,
    methods: methodsText.value.split(',').map((s) => s.trim()).filter(Boolean),
    middlewares: mwText.value.split(',').map((s) => s.trim()).filter(Boolean),
  }
  try {
    if (creating.value) await api.createRoute(body)
    else await api.updateRoute(body.id, body)
    toast.push('路由已热更新')
    drawer.value = false
    await load()
  } catch (err) {
    toast.push(err.message, 'error')
  }
}
async function toggle(r) {
  try {
    await api.toggleRoute(r.id)
    toast.push('状态已切换')
    await load()
  } catch (err) { toast.push(err.message, 'error') }
}
function askDel(r) { delTarget.value = r }
async function confirmDel() {
  try {
    await api.deleteRoute(delTarget.value.id)
    toast.push('已删除')
    delTarget.value = null
    await load()
  } catch (err) { toast.push(err.message, 'error') }
}
onMounted(load)
</script>
