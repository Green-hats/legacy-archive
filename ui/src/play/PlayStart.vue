<template>
  <el-dialog v-model="dialogVisible" :title="playItem.name" center @close="onClose">
    <Artplayer :playItem="playItem"/>
  </el-dialog>
</template>
<script setup>

import {ref} from "vue";
import Artplayer from "./Artplayer.vue";
import {toApiFile} from "@/js/global.js";

let dialogVisible = ref(false)
let playItem = ref({})

let show = (pi) => {
  playItem.value = {...pi};
  playItem.value.src = toApiFile(playItem.value.filename)
  dialogVisible.value = true
}

defineExpose({
  show
})

let onClose = () => {
  dialogVisible.value = false
  playItem.value = {}
}
</script>