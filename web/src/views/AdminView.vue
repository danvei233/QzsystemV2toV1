<template>
  <a-layout class="app-shell">
    <a-layout-sider width="236" class="sider">
      <div class="brand">
        <div class="brand-mark">XP</div>
        <div>
          <div class="brand-title">xiaoheiproxy</div>
          <div class="brand-sub">v2 to v1 gateway</div>
        </div>
      </div>
      <a-menu v-model:selectedKeys="selectedKeys" theme="dark" mode="inline">
        <a-menu-item key="logs">
          <template #icon><DatabaseOutlined /></template>
          请求记录
        </a-menu-item>
        <a-menu-item key="settings">
          <template #icon><SettingOutlined /></template>
          运行参数
        </a-menu-item>
      </a-menu>
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="topbar">
        <div>
          <h2>{{ selectedKeys[0] === "settings" ? "运行参数" : "请求记录" }}</h2>
          <span>完整保留下游入口、上游 v1 调用和响应内容</span>
        </div>
        <a-space>
          <a-button @click="refresh" :loading="loading">
            <template #icon><ReloadOutlined /></template>
            刷新
          </a-button>
          <a-button danger @click="$emit('logout')">
            <template #icon><LogoutOutlined /></template>
            退出
          </a-button>
        </a-space>
      </a-layout-header>

      <a-layout-content class="content">
        <section v-if="selectedKeys[0] === 'logs'" class="panel">
          <div class="toolbar">
            <a-input-search v-model:value="query.keyword" placeholder="搜索 trace、路径、上游 URL、消息" allow-clear @search="loadLogs" />
            <a-select v-model:value="query.success" style="width: 140px" @change="loadLogs">
              <a-select-option value="">全部结果</a-select-option>
              <a-select-option value="true">成功</a-select-option>
              <a-select-option value="false">失败</a-select-option>
            </a-select>
            <a-input v-model:value="query.path" placeholder="/api/v1/info" allow-clear style="width: 220px" @pressEnter="loadLogs" />
          </div>

          <a-table
            row-key="id"
            size="small"
            :columns="columns"
            :data-source="logs"
            :loading="loading"
            :pagination="pagination"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="record.success ? 'success' : 'error'">{{ record.response_status }}</a-tag>
              </template>
              <template v-else-if="column.key === 'path'">
                <a-typography-text :content="record.downstream_path" copyable />
              </template>
              <template v-else-if="column.key === 'upstream'">
                <a-typography-text ellipsis :content="record.upstream_url || '-'" style="max-width: 360px" />
              </template>
              <template v-else-if="column.key === 'created'">
                {{ formatTime(record.created_at) }}
              </template>
              <template v-else-if="column.key === 'cache'">
                <a-tag v-if="record.cache_hit" color="processing">缓存</a-tag>
                <span v-else>-</span>
              </template>
              <template v-else-if="column.key === 'action'">
                <a-button type="link" @click="openDetail(record)">详情</a-button>
              </template>
            </template>
          </a-table>
        </section>

        <section v-else class="panel settings-grid">
          <a-form layout="vertical" :model="settingsForm" @finish="saveSettings">
            <a-row :gutter="16">
              <a-col :xs="24" :lg="12">
                <a-form-item label="管理员账号">
                  <a-input v-model:value="settings.admin_username" disabled />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :lg="12">
                <a-form-item label="新管理员密码">
                  <a-input-password v-model:value="settingsForm.admin_password" placeholder="留空则不修改" autocomplete="new-password" />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :lg="16">
                <a-form-item label="v1 上游 Base URL" name="upstream_base_url" :rules="[{ required: true, message: '请输入上游地址' }]">
                  <a-input v-model:value="settingsForm.upstream_base_url" placeholder="https://panel.example.com/index.php/api/cloud" />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :lg="8">
                <a-form-item label="v1 API Key">
                  <a-input-password v-model:value="settingsForm.upstream_api_key" :placeholder="settings.upstream_api_key_set ? '已设置，留空不修改' : '请输入 API Key'" />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :md="8">
                <a-form-item label="请求超时" name="timeout" :rules="[{ required: true, message: '请输入超时时间' }]">
                  <a-input v-model:value="settingsForm.timeout" placeholder="30s" />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :md="8">
                <a-form-item label="缓存 TTL" name="cache_ttl" :rules="[{ required: true, message: '请输入缓存 TTL' }]">
                  <a-input v-model:value="settingsForm.cache_ttl" placeholder="30s" />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :md="8">
                <a-form-item label="最大请求体">
                  <a-input-number v-model:value="settingsForm.max_body_bytes" :min="1024" :max="10485760" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :md="8">
                <a-form-item label="启用网关 API Key">
                  <a-switch v-model:checked="settingsForm.require_api_key" />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :md="16">
                <a-form-item label="允许的网关 API Key">
                  <a-select
                    v-model:value="settingsForm.accepted_api_keys"
                    mode="tags"
                    :token-separators="[',']"
                    placeholder="输入后回车，支持多个"
                  />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :md="8">
                <a-form-item label="日志保留天数">
                  <a-input-number v-model:value="settingsForm.log_retention_days" :min="1" :max="3650" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :xs="24" :md="8">
                <a-form-item label="日志总大小上限（MB）">
                  <a-input-number v-model:value="settingsForm.log_max_size_mb" :min="1" :max="102400" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :xs="24">
                <a-alert type="info" show-icon>
                  <template #message>运行信息</template>
                  <template #description>
                    当前监听 {{ settings.server_addr || "-" }}，记录存储 {{ settings.database_dsn || "-" }}。日志必须设置保留天数和大小上限，保存后立即裁剪并清空查询缓存。
                  </template>
                </a-alert>
              </a-col>
            </a-row>
            <div class="settings-actions">
              <a-button type="primary" html-type="submit" :loading="savingSettings">保存配置</a-button>
            </div>
          </a-form>
        </section>
      </a-layout-content>
    </a-layout>

    <a-drawer v-model:open="detailOpen" width="70vw" title="请求详情" :destroy-on-close="true">
      <div v-if="detail" class="detail">
        <a-descriptions bordered size="small" :column="2">
          <a-descriptions-item label="Trace">{{ detail.trace_id }}</a-descriptions-item>
          <a-descriptions-item label="客户端">{{ detail.client_ip }}</a-descriptions-item>
          <a-descriptions-item label="路径">{{ detail.downstream_path }}</a-descriptions-item>
          <a-descriptions-item label="耗时">{{ detail.duration_ms }} ms</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="detail.success ? 'success' : 'error'">{{ detail.response_status }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="消息">{{ detail.message || "-" }}</a-descriptions-item>
        </a-descriptions>

        <a-tabs>
          <a-tab-pane key="downstream" tab="下游请求">
            <JsonBlock :content="json({ headers: parse(detail.downstream_headers), body: parse(detail.downstream_body) })" />
          </a-tab-pane>
          <a-tab-pane key="upstream" tab="上游请求">
            <JsonBlock :content="json({ method: detail.upstream_method, url: detail.upstream_url, headers: parse(detail.upstream_headers), body: parse(detail.upstream_body) })" />
          </a-tab-pane>
          <a-tab-pane key="response" tab="响应">
            <JsonBlock :content="json({ status: detail.response_status, headers: parse(detail.response_headers), body: parse(detail.response_body) })" />
          </a-tab-pane>
        </a-tabs>
      </div>
    </a-drawer>
  </a-layout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import { message } from "ant-design-vue";
import { DatabaseOutlined, LogoutOutlined, ReloadOutlined, SettingOutlined } from "@ant-design/icons-vue";
import { api, type AdminSettings, type RequestLog, type UpdateSettingsPayload } from "../api/client";
import JsonBlock from "../components/JsonBlock.vue";

const emit = defineEmits<{ logout: [] }>();

const selectedKeys = ref(["logs"]);
const loading = ref(false);
const logs = ref<RequestLog[]>([]);
const detail = ref<RequestLog | null>(null);
const detailOpen = ref(false);
const settings = ref<Partial<AdminSettings>>({});
const savingSettings = ref(false);
const settingsForm = reactive<UpdateSettingsPayload>({
  admin_password: "",
  upstream_base_url: "",
  upstream_api_key: "",
  timeout: "30s",
  cache_ttl: "30s",
  max_body_bytes: 1048576,
  require_api_key: false,
  accepted_api_keys: [],
  log_retention_days: 7,
  log_max_size_mb: 100
});
const query = reactive({ keyword: "", path: "", success: "" });
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true });

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 80 },
  { title: "路径", key: "path", width: 220 },
  { title: "上游", key: "upstream" },
  { title: "状态", key: "status", width: 90 },
  { title: "缓存", key: "cache", width: 80 },
  { title: "耗时", dataIndex: "duration_ms", key: "duration_ms", width: 90 },
  { title: "消息", dataIndex: "message", key: "message", ellipsis: true },
  { title: "时间", key: "created", width: 180 },
  { title: "操作", key: "action", width: 90 }
];

const loadLogs = async () => {
  loading.value = true;
  try {
    const res = await api.logs({
      keyword: query.keyword || undefined,
      path: query.path || undefined,
      success: query.success || undefined,
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize
    });
    logs.value = res.data.items;
    pagination.total = res.data.total;
  } finally {
    loading.value = false;
  }
};

const loadSettings = async () => {
  const res = await api.settings();
  settings.value = res.data;
  settingsForm.admin_password = "";
  settingsForm.upstream_base_url = res.data.upstream_base_url;
  settingsForm.upstream_api_key = "";
  settingsForm.timeout = res.data.timeout;
  settingsForm.cache_ttl = res.data.cache_ttl;
  settingsForm.max_body_bytes = res.data.max_body_bytes;
  settingsForm.require_api_key = res.data.require_api_key;
  settingsForm.accepted_api_keys = [...(res.data.accepted_api_keys || [])];
  settingsForm.log_retention_days = res.data.log_retention_days || 7;
  settingsForm.log_max_size_mb = res.data.log_max_size_mb || 100;
};

const saveSettings = async () => {
  savingSettings.value = true;
  const passwordChanged = Boolean(settingsForm.admin_password?.trim());
  try {
    await api.updateSettings({
      ...settingsForm,
      admin_password: settingsForm.admin_password?.trim() || undefined,
      upstream_api_key: settingsForm.upstream_api_key?.trim() || undefined,
      accepted_api_keys: settingsForm.accepted_api_keys.map((item) => item.trim()).filter(Boolean)
    });
    message.success(passwordChanged ? "配置已保存，请使用新密码重新登录" : "配置已保存");
    if (passwordChanged) {
      emit("logout");
      return;
    }
    await loadSettings();
  } catch (error: any) {
    message.error(error.response?.data?.error || "保存失败");
  } finally {
    savingSettings.value = false;
  }
};

const refresh = () => {
  if (selectedKeys.value[0] === "settings") {
    loadSettings();
  } else {
    loadLogs();
  }
};

const handleTableChange = (pager: any) => {
  pagination.current = pager.current;
  pagination.pageSize = pager.pageSize;
  loadLogs();
};

const openDetail = async (record: RequestLog) => {
  const res = await api.log(record.id);
  detail.value = res.data;
  detailOpen.value = true;
};

const parse = (raw: string) => {
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch {
    return raw;
  }
};

const json = (value: unknown) => JSON.stringify(value, null, 2);
const formatTime = (value: string) => dayjs(value).format("YYYY-MM-DD HH:mm:ss");

onMounted(() => {
  loadLogs();
  loadSettings();
});
</script>
