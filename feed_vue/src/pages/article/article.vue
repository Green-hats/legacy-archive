<template>
  <div class="wx-page">
    <div class="wx-header">
      <div class="wx-logo">文章详情</div>
    </div>
    <div class="wx-card">
      <span class="main-title">文章内容</span>
      <span class="sub-title">这里展示推荐文章的详细内容</span>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const articleId = route.params.id

onMounted(() => {
  // 可扩展：根据 articleId 获取文章详情
})
</script>

<style scoped>
@import './article.css';
</style>
