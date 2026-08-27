<template>
  <div class="play-grid">
    <a v-for="p in players" :key="p.label" :href="p.url()" class="play-link">
      <el-text>
        <el-icon>
          <img :alt="p.label" class="icon" :src="p.icon"/>
        </el-icon>
        {{ p.label }}
      </el-text>
    </a>
  </div>
</template>

<script setup>
import iconPotPlayer from "../icon/icon-PotPlayer.webp";
import iconVLC from "../icon/icon-VLC.webp";
import iconIINA from "../icon/icon-IINA.webp";
import iconMPV from "../icon/icon-MPV.webp";
import iconInfuse from "../icon/icon-Infuse.png";
import iconDandanPlay from "../icon/icon-DandanPlay.webp";
import iconAnimacX from "../icon/icon-AnimacX.webp";
import iconSenPlayer from "../icon/icon-SenPlayer.webp";

const props = defineProps(['playItem'])

let encodeUrl = (str) => {
  return encodeURIComponent(str);
}

// URL-safe base64 (mpv-handler 要求): / -> _, + -> -, 去掉 = 填充
let b64Url = (str) => {
  const bytes = new TextEncoder().encode(str)
  let bin = ''
  for (const b of bytes) {
    bin += String.fromCharCode(b)
  }
  return btoa(bin).replace(/\//g, '_').replace(/\+/g, '-').replace(/=+$/g, '')
}

const players = [
  {label: "PotPlayer", icon: iconPotPlayer, url: () => `potplayer://${props.playItem.src}`},
  {label: "VLC", icon: iconVLC, url: () => `vlc://${props.playItem.src}`},
  {label: "IINA", icon: iconIINA, url: () => `iina://weblink?url=${encodeUrl(props.playItem.src)}&mpv_force-media-title=${props.playItem.name}`},
  {label: "MPV", icon: iconMPV, url: () => `mpv-handler://play/${b64Url(props.playItem.src)}/?v_title=${b64Url(props.playItem.name)}`},
  {label: "Infuse", icon: iconInfuse, url: () => `infuse://x-callback-url/play?url=${encodeUrl(props.playItem.src)}&filename=${props.playItem.name}`},
  {label: "弹弹Play", icon: iconDandanPlay, url: () => `ddplay:${encodeUrl(props.playItem.src)}|filePath=${props.playItem.name}`},
  {label: "AnimacX", icon: iconAnimacX, url: () => `anix://openVideo/${encodeUrl(props.playItem.src)}`},
  {label: "SenPlayer", icon: iconSenPlayer, url: () => `SenPlayer://x-callback-url/play?url=${encodeUrl(props.playItem.src)}&name=${props.playItem.name}`},
]
</script>

<style scoped>
.play-grid {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px;
  padding: 24px 0;
}

.play-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 56px;
  padding: 0 20px;
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color);
  border: 1px solid var(--el-border-color-light);
  color: var(--el-text-color-primary);
  text-decoration: none;
  cursor: pointer;
  transition: background .2s;
}

.play-link:hover {
  background: var(--el-fill-color-light);
}

.icon {
  height: 20px;
  width: 20px;
}
</style>