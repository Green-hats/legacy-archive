<template>
  <template v-if="notificationConfig['notificationType'] === 'SHELL'">
    <el-form-item label="Shell">
      <el-input v-model:model-value="notificationConfig['shell']" type="textarea"
                :autosize="{ minRows: 2}"
                placeholder='ping 223.5.5.5'/>
    </el-form-item>
  </template>
</template>

<script setup>
let props = defineProps(['notificationConfig', 'config'])
</script>
