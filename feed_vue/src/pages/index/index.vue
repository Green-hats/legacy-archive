<template>
  <div class="wx-page">
    <!-- 顶部区域 -->
    <div class="wx-header">
      <div class="wx-logo">智能喂养</div>
    </div>

    <!-- 主功能区网格布局 -->
    <div class="wx-grid">
      <!-- 主功能卡片 -->
      <div class="wx-card main-card" @click="navigateToFeed">
        <img :src="_imports_0" class="main-icon" />
        <span class="main-title">智能喂食</span>
        <span class="sub-title">定时定量，科学喂养</span>
      </div>
      <!-- 副功能区 -->
      <div class="sub-card-group">
        <div class="wx-card sub-card" @click="navigateToHistory">
        <img :src="_imports_1" class="sub-icon" />
          <span class="sub-title">历史记录</span>
        </div>
        <div class="wx-card sub-card" @click="navigateToCamera">
        <img :src="_imports_2" class="sub-icon" />
          <span class="sub-title">实时影像</span>
        </div>
      </div>
    </div>

    <!-- 推荐内容区域 -->
    <div class="wx-card recommendations">
      <div class="section-header">
        <img :src="_imports_3" class="book-icon" />
        <span class="section-title">个性化推荐</span>
      </div>
      <div class="recommendation-list">
        <div class="recommendation-item" @click="navigateToArticle(1)">
          <div class="content">
            <span class="item-title">Title</span>
            <span class="item-desc"> Sub-title</span>
          </div>
          <span class="info">xxxx人阅读</span>
        </div>
      </div>
    </div>

    <!-- 底部导航栏 -->
    <div class="wx-nav">
      <div class="wx-nav-item active">
        <img :src="_imports_4" class="nav-icon" />
        <span class="nav-text">首页</span>
      </div>
      <div class="wx-nav-item" @click="navigateToProfile">
        <img :src="_imports_5" class="nav-icon" />
        <span class="nav-text">我的</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import _imports_0 from '/src/icons/arm_local.svg'
import _imports_1 from '/src/icons/time-history.svg'
import _imports_2 from '/src/icons/camera.svg'
import _imports_3 from '/src/icons/address-book.svg'
import _imports_4 from '/src/icons/home.svg'
import _imports_5 from '/src/icons/Profile.svg'
const router = useRouter()

function navigateToFeed() {
  router.push({ path: '/feed' })
}
function navigateToHistory() {
  router.push({ path: '/history' })
}
function navigateToCamera() {
  router.push({ path: '/camera' })
}
function navigateToArticle(id) {
  router.push({ path: `/article/${id}` })
}
function navigateToProfile() {
  router.push({ path: '/profile' })
}
</script>

<style scoped>
@import './index.css';
</style>
