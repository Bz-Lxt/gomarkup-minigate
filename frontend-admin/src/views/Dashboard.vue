<template>
  <div>
    <div class="page-head">
      <div>
        <h1>航道总览</h1>
        <p>实时观察数据面流量、节点健康与热更新心跳。</p>
      </div>
      <span class="chip">{{ health }}</span>
    </div>
    <div class="grid-4">
      <div class="card">
        <div class="kicker">QPS</div>
        <div class="metric">{{ stats.qps?.toFixed(2) ?? '0.00' }}</div>
        <div class="spark">
          <i v-for="(n, i) in spark" :key="i" :style="{ height: n + '%' }"></i>
        </div>
      </div>
      <div class="card">
        <div class="kicker">累计请求</div>
        <div class="metric">{{ stats.total_requests ?? 0 }}</div>
      </div>
      <div class="card">
        <div class="kicker">活跃路由</div>
        <div class="metric">{{ stats.active_routes ?? 0 }}</div>
      </div>
      <div class="card">
        <div class="kicker">热更新</div>
        <div class="metric" style="font-size:16px;line-height:1.4">{{ stats.hot_reload?.last_success || '—' }}</div>
        <p class="kicker" style="margin-top:8px">{{ stats.hot_reload?.last_error || '无错误' }}</p>
      </div>
    </div>
    <div class="grid-2" style="margin-top:16px">
      <div class="card">
        <div class="kicker">上游节点</div>
        <table>
          <thead><tr><th>集群</th><th>算法</th><th>健康</th></tr></thead>
          <tbody>
            <tr v-for="u in stats.upstreams || []" :key="u.id">
              <td>{{ u.name }}</td>
              <td class="mono">{{ u.algorithm }}</td>
              <td><span class="led" :class="u.healthy === u.total ? 'on' : 'off'"></span>{{ u.healthy }}/{{ u.total }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="card">
        <div class="kicker">最近错误</div>
        <p v-if="!(stats.recent_errors || []).length" class="kicker">航道清洁，暂无告警。</p>
        <div v-for="(e, i) in stats.recent_errors || []" :key="i" style="margin:10px 0">
          <div class="mono" style="font-size:12px;color:var(--fog)">{{ e.time }}</div>
          <div>{{ e.message }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { api } from '../api'

const stats = ref({})
const spark = ref(Array(24).fill(8))
const health = ref('探测中')
let timer

async function load() {
  try {
    stats.value = await api.stats() || {}
    const q = Math.min(100, (stats.value.qps || 0) * 8 + 8)
    spark.value = spark.value.slice(1).concat(q)
    const h = await api.health()
    health.value = h?.status === 'up' ? 'CONTROL ONLINE' : 'DEGRADED'
  } catch {
    health.value = 'OFFLINE'
  }
}
onMounted(() => { load(); timer = setInterval(load, 2000) })
onUnmounted(() => clearInterval(timer))
</script>
