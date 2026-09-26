// ECharts 按需引入统一入口。
//
// 原先各组件使用 `import * as echarts from 'echarts'` 整包引入，导致构建产物
// echarts-*.js ≈ 1.1MB（gzip ≈ 379KB）。此处改为 tree-shaking 友好的
// `echarts/core` + 按需图表/组件注册，只引入实际用到的模块。
//
// 注册清单（依据 src/views/reports/index.vue 的 6 张图表 option，经 grep 全量排查）：
//   图表   ：BarChart（柱状图：进销存汇总 / 特殊药品使用 / 调配工作量 / 拆零统计）、
//            PieChart（饼图：效期分析 / 患者费用）
//   组件   ：GridComponent（直角坐标系 xAxis/yAxis）、
//            TooltipComponent（tooltip trigger: axis|item + valueFormatter）、
//            LegendComponent（legend）
//   渲染器 ：CanvasRenderer
//
// 未使用故未注册：LineChart、TitleComponent、DataZoomComponent、ToolboxComponent、
//   MarkLineComponent、MarkPointComponent、MarkAreaComponent、DatasetComponent、
//   TransformComponent、AxisPointerComponent、GraphicComponent 等。
import { init, use } from 'echarts/core'
import { BarChart, PieChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsOption, EChartsType } from 'echarts/types/dist/shared'

use([BarChart, PieChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer])

export { init }
export type { EChartsOption, EChartsType }
