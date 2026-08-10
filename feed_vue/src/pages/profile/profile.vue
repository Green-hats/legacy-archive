<template>
  <div class="wx-page">
    <div class="wx-header">
      <div class="wx-logo">我的</div>
    </div>
    <div class="wx-card">
      <span class="main-title">个人中心</span>
      <span class="sub-title">这里展示用户信息和设置</span>
    </div>
    <!-- 底部导航栏 -->
    <div class="wx-nav">
      <div class="wx-nav-item" @click="navigateToIndex">
        <img src="/src/icons/home.svg" class="nav-icon" />
        <span class="nav-text">首页</span>
      </div>
      <div class="wx-nav-item active">
        <img src="/src/icons/Profile.svg" class="nav-icon" />
        <span class="nav-text">我的</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
const router = useRouter()

function navigateToIndex() {
  router.push({ path: '/' })
}
// 可扩展：个人中心相关逻辑
</script>

<style scoped>
@import './profile.css';
</style>
