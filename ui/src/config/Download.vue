<template>
  <el-form @submit.prevent label-width="auto"
           class="full-width">
    <el-form-item label="下载工具">
      <el-select v-model:model-value="props.config.downloadToolType">
        <el-option v-for="item in downloadSelect"
                   :key="item"
                   :label="item"
                   :value="item"/>
      </el-select>
    </el-form-item>
    <el-form-item label="115 Cookie">
      <el-input v-model.trim="props.config.pan115Cookie"
                type="textarea"
                :autosize="{minRows: 2}"
                placeholder="UID=...; CID=...; SEID=...; KID=..."
                show-password/>
      <br/>
      <el-text class="mx-1" size="small">
        115 为云端离线下载:番剧将下载到 115 网盘,播放走网盘链接。填入浏览器登录 115 后的 Cookie
      </el-text>
    </el-form-item>
    <el-form-item>
      <div class="download-test-button">
        <el-button @click="downloadLoginTest" bg text :loading="downloadLoginTestLoading" icon="Odometer">测试
        </el-button>
      </div>
    </el-form-item>
    <el-form-item label="保存位置">
      <el-input v-model.trim="props.config['downloadPathTemplate']"/>
      <div class="full-width margin-top-4" v-if="!testPathTemplate(props.config['downloadPathTemplate'])">
        <el-alert
            type="warning"
            show-icon
            :closable="false"
        >
          <template #title>
            你的 保存位置 并未按照模版填写, 可能会遇到下载位置错误
          </template>
        </el-alert>
      </div>
    </el-form-item>
    <el-form-item label="剧场版保存位置">
      <el-input v-model.trim="props.config['ovaDownloadPathTemplate']"/>
      <div class="full-width margin-top-4" v-if="!testPathTemplate(props.config['ovaDownloadPathTemplate'])">
        <el-alert
            type="warning"
            show-icon
            :closable="false"
        >
          <template #title>
            你的 剧场版保存位置 并未按照模版填写, 可能会遇到下载位置错误
          </template>
        </el-alert>
      </div>
    </el-form-item>
    <el-form-item label="失败重试次数">
      <el-input-number v-model:model-value="props.config['downloadRetry']" :max="100" :min="3"/>
    </el-form-item>
    <el-form-item label="同时下载限制">
      <div>
        <el-input-number v-model:model-value="props.config.downloadCount" :min="0"/>
        <div>
          设置为时 0 不做限制
        </div>
      </div>
    </el-form-item>
    <el-form-item label="延迟下载">
      <el-input-number v-model:model-value="props.config.delayedDownload" :min="0">
        <template #suffix>
          <span>分钟</span>
        </template>
      </el-input-number>
    </el-form-item>
    <el-form-item label="优先保留">
      <div class="full-width">
        <el-switch v-model:model-value="props.config.priorityKeywordsEnable"/>
        <div>
          <el-text class="mx-1" size="small">
            启用多文件种子的文件优先保留过滤
          </el-text>
        </div>
        <div v-if="props.config.priorityKeywordsEnable">
          <PrioKeys
              v-model:keywords="props.config.priorityKeywords"
              :import-global="false"
              :show-text="true"
          />
        </div>
      </div>
    </el-form-item>
    <el-form-item label="自定义标签">
      <custom-tags :config="props.config"/>
    </el-form-item>
  </el-form>
</template>

<script setup>
import {ref} from "vue";
import {ElMessage, ElText} from "element-plus";
import PrioKeys from "@/config/PrioKeys.vue";
import CustomTags from "@/config/CustomTags.vue";
import * as http from "@/js/http.js";

const downloadSelect = ref([
  '115'
])

const downloadLoginTestLoading = ref(false)
const downloadLoginTest = () => {
  downloadLoginTestLoading.value = true
  http.downloadLoginTest(props.config)
      .then(res => {
        ElMessage.success(res.message)
      })
      .finally(() => {
        downloadLoginTestLoading.value = false
      })
}

let testPathTemplate = (path) => {
  return new RegExp('\\$\{[A-z]+\}').test(path);
}

let props = defineProps(['config'])
</script>

<style scoped>
.download-test-button {
  display: flex;
  width: 100%;
  justify-content: end;
}
</style>