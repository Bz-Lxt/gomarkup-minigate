# MiniGate Design Spec

> 审美方向：**铜火夜航台（Copper Night Bridge）**
> 对标机房值班台 / 航道调度屏，而不是通用 SaaS 后台。

## 1. 概念

MiniGate 的操作者是网关值班员。界面必须让「流量是否在动、节点是否活着、配置是否刚热更新」一眼可读。视觉记忆点是 **暗墨底 + 铜火强调色 + 等宽指标**，而不是紫色渐变或默认 Element 皮肤。

## 2. 色彩

| Token | Hex | 用途 |
|---|---|---|
| `--ink` | `#07090D` | 画布底色 |
| `--ink-2` | `#10141C` | 侧栏 / 卡片 |
| `--ink-3` | `#181E2A` | 悬浮层 |
| `--line` | `#2A3344` | 分割线 |
| `--copper` | `#E8A04A` | 主强调 / 活动态 |
| `--signal` | `#3DDC97` | 健康 / 成功 |
| `--rose` | `#F07178` | 危险 / 限流拒绝 |
| `--fog` | `#9AA6B8` | 次级文字 |
| `--paper` | `#E8EDF5` | 主文字 |

禁止：Inter、Roboto、紫白渐变、原生 `alert/confirm`。

## 3. 字体

- 展示 / 导航：`Sora`（Google Fonts）
- 正文：`Manrope`
- 指标 / 路径 / JSON：`IBM Plex Mono`

## 4. 布局

- 左侧 248px 固定航道导航（平板 / 手机折叠为顶栏）
- 主区 `w-full`，无 `max-w-*` 限宽
- 768px：侧栏收起；480px：表单单列、表格横向滚动

## 5. 组件

- **Toast**：右上角，可手动关闭，5s 自动消失
- **Modal**：危险删除必须二次确认
- **Form**：必填 `*` + 字段下方红字；保存前统一 `validate()`
- **Select**：自定义箭头，禁止原生 appearance
- **LED**：上游节点健康用 8px 圆点呼吸

## 6. 页面

1. Dashboard — QPS 火花条、路由数、节点健康、最近错误
2. Routes — 表格 + 抽屉表单
3. Upstreams — 节点权重可视化条
4. Middlewares — 三张插件卡 + JSON Schema 表单
5. Config — 热更新状态 + YAML 只读预览
