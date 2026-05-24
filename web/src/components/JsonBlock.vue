<template>
  <pre class="json-block">{{ content }}</pre>
</template>

<script setup lang="ts">
defineProps<{ content: string }>();
</script>

<style scoped>
.json-block {
  margin: 0;
  max-height: 420px;
  overflow: auto;
  padding: 14px;
  background: #15191f;
  color: #d8dee9;
  border: 1px solid #262c35;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>

