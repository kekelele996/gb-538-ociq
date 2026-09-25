<script setup lang="ts">
import { AudioWaveform, CircleGauge, ShieldCheck } from '@lucide/vue'
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { errorMessage } from '../api/client'
import { useAuthStore } from '../stores/auth-store'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const form = reactive({ username: 'engineer', password: 'engineer123' })
const error = ref('')

async function submit() {
  error.value = ''
  try {
    await auth.login(form.username, form.password)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/points'
    await router.replace(redirect)
  } catch (reason) {
    error.value = errorMessage(reason)
  }
}
</script>

<template>
  <main class="login-shell">
    <section class="login-context">
      <div class="login-brand"><AudioWaveform :size="20" /> NoiseTrace 噪声源贡献归因台</div>
      <div class="login-title">
        <p class="eyebrow">OCTAVE EVIDENCE / OFFLINE NNLS</p>
        <h1>把每一分贝<br />还给它的来源</h1>
        <p>固定倍频程、能量域背景扣除和版本化非负最小二乘，将监测结果变成可复核的声源贡献证据。</p>
      </div>
      <div class="spectrum-board" aria-hidden="true">
        <span v-for="height in [38, 55, 77, 68, 49, 34, 24, 16]" :key="height" :style="{ height: `${height}%` }" />
      </div>
      <p class="login-boundary"><ShieldCheck :size="14" /> 离线分析，不连接设备，不替代专业声学报告</p>
    </section>
    <section class="login-panel">
      <el-form class="login-form" label-position="top" @submit.prevent="submit">
        <CircleGauge :size="23" />
        <p class="eyebrow">SECURE WORKSPACE</p>
        <h2>进入归因工作台</h2>
        <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
        <el-form-item label="账号"><el-input v-model="form.username" autocomplete="username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password autocomplete="current-password" /></el-form-item>
        <el-button type="primary" native-type="submit" :loading="auth.busy">登录</el-button>
        <p class="credential-note">演示：engineer / engineer123</p>
      </el-form>
    </section>
  </main>
</template>
