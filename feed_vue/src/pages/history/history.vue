<template>
  <div class="wx-page">
    <div class="wx-header">
      <div style="background: none; color: #2a73d9; font-weight: 800; font-size: 24px; padding-left: 0;">喂食记录历史</div>
    </div>
    <div class="wx-card">
      <div class="statistics-container">
        <div class="stat-item">
          <div class="stat-value">{{ statistics.today_count }}</div>
          <div class="stat-label">今日喂食</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ statistics.avg_score }}</div>
          <div class="stat-label">平均评分</div>
        </div>
      </div>
      <div v-if="loading" class="state-container loading-state">加载中...</div>
      <div v-if="error" class="state-container error-state">{{ error }}</div>
      <div v-if="feedings.length" class="feeding-list">
        <div v-for="item in feedings" :key="item.record_id" class="feeding-item-card">
          <div class="feeding-item-header">
            <div class="person-icon">{{ item.person_name && item.person_name[0] }}</div>
            <span class="person-name">{{ item.person_name }}</span>
            <span class="feeding-time">{{ item.feeding_time }}</span>
          </div>
          <div class="feeding-item-body">
            <div class="food-row">
              <div class="row-icon">
                <img src="/src/icons/food.svg" class="sub-icon" />
              </div>
              <div class="row-content">
                <div class="food-main">
                  <span class="food-name">{{ item.food_name }}</span>
                  <span v-if="item.food_category" class="food-category-tag">{{ item.food_category }}</span>
                </div>
              </div>
            </div>
            <div class="location-row" v-if="item.location_name">
              <div class="row-icon">
                <img src="/src/icons/order-location.svg" class="sub-icon" />
              </div>
              <span class="location-name">{{ item.location_name }}</span>
            </div>
            <div class="portion-row" v-if="item.portion_size">
              <div class="row-icon">
                <img src="/src/icons/number.svg" class="sub-icon" />
              </div>
              <div class="portion-label">分量</div>
              <div class="portion-value">{{ item.portion_size }} 克</div>
            </div>
            <div class="notes-row" v-if="item.notes">
               <div class="row-icon">
                <img src="/src/icons/score.svg" class="sub-icon" />
              </div>
              <div class="notes-label">备注</div>
              <div class="notes-value">{{ item.notes }}</div>
            </div>
            <div class="score-row" v-if="item.score">
              <div class="score-label">评分</div>
              <div class="score-row-content">
                <span class="star-container">
                  <span v-for="n in 5" :key="n" class="star-icon" :class="{ active: n <= Math.round(item.score) }">★</span>
                </span>
                <span class="score-value">{{ item.score }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const feedings = ref([])
const loading = ref(false)
const error = ref('')
const statistics = ref({ today_count: 0, avg_score: '-' })

async function fetchFeedings() {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch('http://localhost:3001/api/feedings')
    if (res.ok) {
      feedings.value = await res.json()
    } else {
      error.value = '获取数据失败'
    }
  } catch (e) {
    error.value = '网络错误，无法获取数据'
  } finally {
    loading.value = false
  }
}

async function fetchStatistics() {
  try {
    const res = await fetch('http://localhost:3001/api/stats')
    if (res.ok) {
      const data = await res.json()
      statistics.value.today_count = data.today_count || 0
      statistics.value.avg_score = data.avg_rating !== undefined ? data.avg_rating : '-'
    } else {
      statistics.value = { today_count: 0, avg_score: '-' }
    }
  } catch (e) {
    statistics.value = { today_count: 0, avg_score: '-' }
  }
}

onMounted(() => {
  fetchFeedings()
  fetchStatistics()
})
</script>

<style scoped>
@import './history.css';
</style>
