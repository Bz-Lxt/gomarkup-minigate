<template>
  <div>
    <div class="page-head">
      <div>
        <h1>上游集群</h1>
        <p>轮询 / 随机 / 平滑加权。节点权重即时参与调度。</p>
      </div>
      <button class="btn" @click="openCreate">新建上游</button>
    </div>
    <div class="card table-wrap">
      <table>
        <thead><tr><th>名称</th><th>算法</th><th>超时</th><th>节点</th><th></th></tr></thead>
        <tbody>
          <tr v-for="u in rows" :key="u.id">
            <td>{{ u.name }}</td>
            <td class="mono">{{ u.algorithm }}</td>
            <td>{{ u.timeout_ms }} ms</td>
            <td>
              <div v-for="n in u.nodes" :key="n.target" style="margin-bottom:8px">
                <div class="mono" style="font-size:12px">{{ n.target }} · w={{ n.weight }}</div>
                <div class="weight-bar"><span :style="{ width: bar(n, u) + '%' }"></span></div>
              </div>
            </td>
            <td class="row-actions">
              <button class="btn ghost" @click="edit(u)">编辑</button>
              <button class="btn danger" @click="askDel(u)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="drawer" class="drawer-mask" @click.self="drawer=false">
      <div class="drawer">
        <h2>{{ creating ? '新建上游' : '编辑上游' }}</h2>
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
          <label>负载均衡</label>
          <select v-model="form.algorithm">
            <option value="round_robin">轮询 round_robin</option>
            <option value="random">随机 random</option>
            <option value="weighted_rr">加权 weighted_rr</option>
            <option value="least_conn">最少连接 least_conn</option>
          </select>
        </div>
        <div class="field">
          <label>超时 (ms)</label>
          <input v-model.number="form.timeout_ms" type="number" min="100" />
        </div>
        <div class="field">
          <label>失败摘除阈值</label>
          <input v-model.number="form.fail_threshold" type="number" min="1" />
        </div>
        <div class="field">
          <label>节点 JSON <span class="req">*</span></label>
          <textarea v-model="nodesText" />
          <div v-if="errors.nodes" class="err">{{ errors.nodes }}</div>
        </div>
        <button class="btn" @click="save">保存</button>
        <button class="btn ghost" style="margin-left:8px" @click="drawer=false">取消</button>
      </div>
    </div>

    <div v-if="delTarget" class="modal-mask">
      <div class="modal">
        <h3>删除上游 {{ delTarget.name }}？</h3>
        <p>若仍被路由引用，服务端会拒绝删除。</p>
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
const drawer = ref(false)
const creating = ref(true)
const form = ref({})
const nodesText = ref('[]')
const errors = ref({})
const delTarget = ref(null)

function bar(n, u) {
  const sum = (u.nodes || []).reduce((a, x) => a + (x.weight || 1), 0) || 1
  return Math.round(((n.weight || 1) / sum) * 100)
}
async function load() { rows.value = await api.upstreams() || [] }
function openCreate() {
  creating.value = true
  form.value = { id: '', name: '', algorithm: 'round_robin', timeout_ms: 5000, fail_threshold: 3 }
  nodesText.value = JSON.stringify([{ target: 'http://upstream-a:9001', weight: 1 }], null, 2)
  errors.value = {}
  drawer.value = true
}
function edit(u) {
  creating.value = false
  form.value = { ...u }
  nodesText.value = JSON.stringify(u.nodes || [], null, 2)
  errors.value = {}
  drawer.value = true
}
function validate() {
  const e = {}
  if (!form.value.id?.trim()) e.id = 'ID 必填'
  if (!form.value.name?.trim()) e.name = '名称必填'
  let nodes
  try {
    nodes = JSON.parse(nodesText.value)
    if (!Array.isArray(nodes) || !nodes.length) e.nodes = '至少 1 个节点'
    else if (nodes.some((n) => !/^https?:\/\//.test(n.target || ''))) e.nodes = 'target 必须是 http(s) URL'
  } catch {
    e.nodes = '节点 JSON 无法解析'
  }
  errors.value = e
  return { ok: !Object.keys(e).length, nodes }
}
async function save() {
  const { ok, nodes } = validate()
  if (!ok) { toast.push('请修正表单错误', 'error'); return }
  const body = { ...form.value, nodes }
  try {
    if (creating.value) await api.createUpstream(body)
    else await api.updateUpstream(body.id, body)
    toast.push('上游已热更新')
    drawer.value = false
    await load()
  } catch (err) { toast.push(err.message, 'error') }
}
function askDel(u) { delTarget.value = u }
async function confirmDel() {
  try {
    await api.deleteUpstream(delTarget.value.id)
    toast.push('已删除')
    delTarget.value = null
    await load()
  } catch (err) { toast.push(err.message, 'error') }
}
onMounted(load)
</script>
