<template>
  <el-form @submit.prevent label-width="auto"
           class="full-width">
    <el-form-item label="Api">
      <el-button icon="DocumentCopy" @click="copyEmbyApi">复制 emby 自动点格子 api</el-button>
      <el-button icon="DocumentCopy" @click="copyIcs">复制 ics</el-button>
    </el-form-item>
    <el-form-item label="Mikan">
      <el-input v-model:model-value="props.config.mikanHost" placeholder="https://mikanani.me"/>
    </el-form-item>
    <el-form-item label="最大日志条数">
      <div class="width-150">
        <el-select v-model:model-value="props.config.logsMax">
          <el-option v-for="it in [128,256,512]" :key="it" :label="it" :value="it"/>
        </el-select>
      </div>
    </el-form-item>
    <el-form-item label="DEBUG">
      <el-switch v-model:model-value="props.config.debug"/>
    </el-form-item>
    <el-form-item label="缓存">
      <div class="full-width">
        <div>
          <el-button :loading="clearCacheLoading" bg icon="Delete" @click="clearCache">清理</el-button>
        </div>
        <div>
          <el-text class="mx-1" size="small">
            清理现在不被使用的缓存
          </el-text>
        </div>
      </div>
    </el-form-item>
  </el-form>
</template>

<script setup>
import {ElMessage, ElText} from "element-plus";
import {ref} from "vue";
import * as http from "@/js/http.js";
import {getBaseUrl} from "@/js/global.js";

let clearCacheLoading = ref(false)
let clearCache = () => {
  clearCacheLoading.value = true
  http.clearCache()
      .then(res => {
        ElMessage.success(res.message);
      })
      .finally(() => {
        clearCacheLoading.value = false
      })
}

let copyEmbyApi = () => {
  let url = `${getBaseUrl()}api/embyWebHook?api-key=${props.config.apiKey}`;
  copy(url)
}

let copyIcs = () => {
  let url = `${getBaseUrl()}api/calendar.ics?api-key=${props.config.apiKey}`;
  copy(url)
}

let copy = (v) => {
  const input = document.createElement('input');
  input.value = v
  document.body.appendChild(input);
  input.select();
  document.execCommand('copy');
  document.body.removeChild(input);
  ElMessage.success('已复制')
}

let props = defineProps(['config'])
</script>

