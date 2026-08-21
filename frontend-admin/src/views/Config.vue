<template>
  <div>
    <div class="page-head">
      <div>
        <h1>配置中心</h1>
        <p>单一事实来源：YAML 文件。校验失败保留旧表，并在此显示错误。</p>
      </div>
      <span class="chip">{{ status.source || 'file' }}</span>
    </div>
    <div class="grid-2">
      <div class="card">
        <div class="kicker">热更新状态</div>
        <p>最后成功：<span class="mono">{{ status.last_success || '—' }}</span></p>
        <p>最近错误：<span class="mono" :style="{ color: status.last_error ? 'var(--rose)' : 'var(--signal)' }">{{ status.last_error || '无' }}</span></p>
      </div>
      <div class="card">
        <div class="kicker">监听说明</div>
        <p>Admin 写操作会原子替换 <code>config/gateway.yaml</code>，fsnotify 300ms 防抖后重建 Radix Tree 与负载均衡器，在途请求不受影响。</p>
      </div>
    </div>
    <div class="card" style="margin-top:16px">
      <div class="kicker">当前生效快照</div>
      <pre class="yaml">{{ pretty }}</pre>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'

const cfg = ref({})
const status = ref({})
const pretty = computed(() => JSON.stringify(cfg.value, null, 2))

onMounted(async () => {
  cfg.value = await api.config() || {}
  status.value = await api.configStatus() || {}
})
</script>
