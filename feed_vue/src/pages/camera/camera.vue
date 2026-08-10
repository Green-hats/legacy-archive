<template>
  <div class="wx-page">
    <div class="wx-header">
      <div class="wx-logo">实时影像</div>
    </div>
    <div class="wx-card">
      <span class="main-title">实时监控</span>
      <span class="sub-title">可在此查看实时画面</span>
      <div class="page-body">
        <div class="mainContainer">
          <video ref="videoElement" class="centeredVideo" controls autoplay width="1024" height="576">
            Your browser is too old which doesn't support HTML5 video.
          </video>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import flvjs from 'flv.js'

const videoElement = ref(null)
let flvPlayer = null

onMounted(() => {
  if (flvjs.isSupported() && videoElement.value) {
    flvPlayer = flvjs.createPlayer({
      type: 'flv',
      isLive: true,
      url: 'http://192.168.100.162:8080/live/livestream.flv', // 这里填你的直播源flv格式
    })
    flvPlayer.attachMediaElement(videoElement.value)
    flvPlayer.load()
    videoElement.value.play()
  }
})

onBeforeUnmount(() => {
  if (flvPlayer) {
    flvPlayer.pause()
    flvPlayer.unload()
    flvPlayer.detachMediaElement()
    flvPlayer.destroy()
    flvPlayer = null
  }
})
</script>

<style scoped>
@import './camera.css';
</style>
