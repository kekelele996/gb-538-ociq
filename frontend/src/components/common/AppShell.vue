<script setup lang="ts">
import { Activity, AudioLines, CircleGauge, Factory, LogOut, MapPinned, ScrollText, Waves } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../../hooks/useAuth'

const router = useRouter()
const { auth, canAudit } = useAuth()
const links = [
  { to: '/points', label: '监测点', icon: MapPinned },
  { to: '/measurements', label: '测量工作台', icon: Activity },
  { to: '/sources', label: '声源谱', icon: Factory },
  { to: '/attribution', label: '贡献归因', icon: CircleGauge },
]

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="app-shell">
    <a class="skip-link" href="#main-content">跳到主要内容</a>
    <aside class="app-sidebar">
      <div class="brand-block" aria-label="NoiseTrace 噪声源贡献归因台">
        <span class="brand-mark"><Waves :size="20" /></span>
        <div><strong>NoiseTrace</strong><span>噪声源贡献归因台</span></div>
      </div>
      <nav aria-label="主导航">
        <RouterLink v-for="link in links" :key="link.to" :to="link.to" class="nav-link">
          <component :is="link.icon" :size="17" /><span>{{ link.label }}</span>
        </RouterLink>
        <RouterLink v-if="canAudit" to="/audit" class="nav-link"><ScrollText :size="17" /><span>审计中心</span></RouterLink>
      </nav>
      <div class="identity-block">
        <span>{{ auth.user?.display_name }}</span>
        <strong>{{ auth.user?.role }}</strong>
        <el-tooltip content="退出登录" placement="right">
          <el-button text circle aria-label="退出登录" @click="logout"><LogOut :size="17" /></el-button>
        </el-tooltip>
      </div>
    </aside>
    <main id="main-content" class="app-main">
      <div class="context-strip"><span>OFFLINE ACOUSTIC EVIDENCE</span><strong>ALGORITHM octave-nnls-v1.0.0</strong></div>
      <div class="boundary-strip">
        <AudioLines :size="17" />
        <strong>离线决策边界</strong>
        <span>结果不替代法定测量或专业声学报告，也不连接或控制采集器、生产设备与降噪装置。</span>
      </div>
      <slot />
    </main>
  </div>
</template>
