<template>
  <div class="http-request-bar">
    <div class="method-badge" :class="methodClass">{{ methodLabel }}</div>
    <div class="url">{{ url || "-" }}</div>
    <div v-if="status !== undefined || duration !== undefined" class="status-info">
      <a-tag v-if="status !== undefined" :color="statusColor">{{ status }}</a-tag>
      <span v-if="duration !== undefined" class="duration">{{ duration }}ms</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(defineProps<{
  method: string;
  url?: string;
  status?: number;
  duration?: number;
}>(), {
  method: "GET",
  url: ""
});

const methodLabel = computed(() => String(props.method || "GET").toUpperCase());
const methodClass = computed(() => {
  const method = methodLabel.value.toLowerCase();
  return ["get", "post", "put", "patch", "delete"].includes(method) ? `method-${method}` : "method-default";
});
const statusColor = computed(() => {
  const status = props.status ?? 0;
  if (status >= 200 && status < 300) return "success";
  if (status >= 300 && status < 400) return "processing";
  if (status >= 400 && status < 500) return "warning";
  if (status >= 500) return "error";
  return "default";
});
</script>

<style scoped>
.http-request-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
}

.method-badge {
  min-width: 56px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 700;
  text-align: center;
}

.method-get {
  background: rgba(22, 119, 255, 0.12);
  color: #1677ff;
}

.method-post {
  background: rgba(82, 196, 26, 0.12);
  color: #389e0d;
}

.method-put,
.method-patch {
  background: rgba(250, 140, 22, 0.12);
  color: #d46b08;
}

.method-delete {
  background: rgba(255, 77, 79, 0.12);
  color: #cf1322;
}

.method-default {
  background: rgba(0, 0, 0, 0.08);
  color: rgba(0, 0, 0, 0.72);
}

.url {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: Consolas, "Liberation Mono", Menlo, monospace;
  font-size: 12px;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.duration {
  color: rgba(0, 0, 0, 0.48);
  font-family: Consolas, "Liberation Mono", Menlo, monospace;
  font-size: 12px;
}
</style>
