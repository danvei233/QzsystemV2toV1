<template>
  <div class="http-headers-table">
    <a-table
      :columns="columns"
      :data-source="dataSource"
      :pagination="false"
      size="small"
      row-key="key"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'key'">
          <code class="header-key">{{ record.key }}</code>
        </template>
        <template v-else-if="column.key === 'value'">
          <code class="header-value">{{ record.value }}</code>
        </template>
      </template>
    </a-table>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{ headers: Record<string, string> | null }>();

const columns = [
  { title: "Key", dataIndex: "key", key: "key", width: "32%" },
  { title: "Value", dataIndex: "value", key: "value" }
];

const dataSource = computed(() => {
  if (!props.headers) return [];
  return Object.entries(props.headers).map(([key, value]) => ({
    key,
    value: String(value)
  }));
});
</script>

<style scoped>
.http-headers-table {
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  overflow: hidden;
}

.http-headers-table :deep(.ant-table-thead > tr > th) {
  background: rgba(0, 0, 0, 0.03);
  font-size: 12px;
  font-weight: 600;
}

.header-key,
.header-value {
  font-family: Consolas, "Liberation Mono", Menlo, monospace;
  font-size: 12px;
  line-height: 1.6;
}

.header-key {
  color: #1677ff;
}

.header-value {
  color: rgba(0, 0, 0, 0.78);
  word-break: break-all;
}
</style>

