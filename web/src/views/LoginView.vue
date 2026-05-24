<template>
  <main class="login-shell">
    <section class="login-panel">
      <div>
        <div class="eyebrow">xiaoheiproxy</div>
        <h1>API 网关控制台</h1>
        <p>查看 v2 到 v1 的上下游请求、响应与缓存命中。</p>
      </div>

      <a-form layout="vertical" :model="form" @finish="submit">
        <a-form-item label="管理员" name="username" :rules="[{ required: true, message: '请输入管理员账号' }]">
          <a-input v-model:value="form.username" autocomplete="username" />
        </a-form-item>
        <a-form-item label="密码" name="password" :rules="[{ required: true, message: '请输入密码' }]">
          <a-input-password v-model:value="form.password" autocomplete="current-password" />
        </a-form-item>
        <a-button type="primary" html-type="submit" block :loading="loading">登录</a-button>
      </a-form>
    </section>
  </main>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { message } from "ant-design-vue";
import { api } from "../api/client";

const emit = defineEmits<{ "logged-in": [token: string] }>();
const loading = ref(false);
const form = reactive({ username: "admin", password: "" });

const submit = async () => {
  loading.value = true;
  try {
    const res = await api.login(form);
    emit("logged-in", res.data.token);
  } catch (error: any) {
    message.error(error.response?.data?.error || "登录失败");
  } finally {
    loading.value = false;
  }
};
</script>

