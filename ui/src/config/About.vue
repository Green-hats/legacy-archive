<template>
  <div class="about-container">
    <div class="flex about-header">
      <img alt="icon.svg" height="80" src="../../public/icon.svg" width="80"/>
      <div>
        <h1>ANI-RSS (Go 二创版)</h1>
        <el-text class="mx-1 cursor-pointer" size="small">
          v{{ about.version }}
        </el-text>
        <br/>
        <el-text class="mx-1" size="small" type="info">
          Go 重写版 · 网盘(115)下载 · 外置播放器播放
        </el-text>
      </div>
    </div>

    <el-divider/>

    <div class="about-section">
      <h3>项目简介</h3>
      <el-text>
        基于 RSS 自动追番、订阅、下载、刮削、洗版的工具。后端用 Go 重写,前端基于上游 ani-rss-ui 二创定制。
        下载走网盘(115 云端离线下载),播放不使用在线播放器,全部通过外置播放器跳转。
      </el-text>
    </div>

    <div class="about-section">
      <h3>功能</h3>
      <ul class="about-list">
        <li><strong>下载</strong>:115 云端离线下载(浏览器 Cookie)</li>
        <li><strong>播放</strong>:PotPlayer / VLC / IINA / MPV / Infuse / 弹弹Play / AnimacX / SenPlayer 跳转播放</li>
        <li><strong>刮削</strong>:TMDB(NFO/poster/fanart)、Bangumi 搜索/评分</li>
        <li><strong>通知</strong>:Telegram / Bark / ServerChan / WebHook / Shell / 邮件 / Emby</li>
        <li><strong>其他</strong>:RSS 抓取、重命名、缺集/摸鱼检测、备用RSS 洗版</li>
      </ul>
    </div>

    <div class="about-section">
      <h3>MPV 播放说明</h3>
      <el-text>
        MPV 按钮通过 <code>mpv-handler://</code> 协议唤起本机 mpv。请先安装
        <el-link type="primary" href="https://github.com/akiirui/mpv-handler" target="_blank">
          akiirui/mpv-handler
        </el-link>
        ,并在 Firefox 的 <code>about:config</code> 中新建布尔值
        <code>network.protocol-handler.expose.mpv-handler = false</code>。
      </el-text>
    </div>

    <div class="about-section">
      <h3>免责声明</h3>
      <el-text type="info">
        本工具为中立性技术辅助工具,请遵守当地法律法规,勿用于盗版传播。
      </el-text>
    </div>

    <el-divider/>

    <div v-loading.fullscreen.lock="actionLoading" class="flex about-actions">
      <popconfirm title="你确定要退出吗?" @confirm="logout">
        <template #reference>
          <el-button type="danger" bg text icon="Back">
            退出
          </el-button>
        </template>
      </popconfirm>
      <div class="about-action-spacer"></div>
      <popconfirm title="你确定重启吗?" @confirm="stop(0)">
        <template #reference>
          <el-button bg icon="RefreshRight" text type="warning">重启</el-button>
        </template>
      </popconfirm>
      <div class="about-action-spacer"></div>
      <popconfirm title="你确定关闭吗?" @confirm="stop(1)">
        <template #reference>
          <el-button bg icon="SwitchButton" text type="danger">关闭</el-button>
        </template>
      </popconfirm>
    </div>
  </div>
</template>

<script setup>
import {onMounted, ref} from "vue";
import {ElMessage} from "element-plus";
import Popconfirm from "@/other/Popconfirm.vue";

import {authorization} from "@/js/global.js";
import * as http from "@/js/http.js";

const actionLoading = ref(false)

const stop = (status) => {
  actionLoading.value = true
  http.stop(status)
      .then(res => {
        ElMessage.success(res.message)
        setTimeout(() => {
          authorization.value = ''
          location.reload()
        }, 5000)
      })
      .finally(() => {
        actionLoading.value = false
      })
}

const about = ref({
  'version': '',
  'latest': '',
  'update': false,
  'markdownBody': ''
})

onMounted(() => {
  http.about()
      .then(res => {
        about.value = res.data
      })
})

let logout = () => {
  authorization.value = ''
  location.reload()
}

let props = defineProps(['config'])
</script>

<style scoped>
.about-container {
  width: 100%;
  flex-flow: column;
  padding: 8px 12px;
}

.about-header {
  align-items: end;
}

.cursor-pointer {
  cursor: pointer;
}

.about-section {
  margin-bottom: 16px;
}

.about-section h3 {
  margin: 0 0 6px;
}

.about-list {
  margin: 0;
  padding-left: 18px;
  line-height: 1.9;
}

.about-actions {
  justify-content: center;
  margin-bottom: 8px;
}

.about-action-spacer {
  margin: 6px;
}
</style>