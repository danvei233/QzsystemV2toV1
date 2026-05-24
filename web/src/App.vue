<template>
  <LoginView v-if="!authed" @logged-in="handleLoggedIn" />
  <AdminView v-else @logout="handleLogout" />
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import LoginView from "./views/LoginView.vue";
import AdminView from "./views/AdminView.vue";
import { api } from "./api/client";

const authed = ref(Boolean(localStorage.getItem("xhp_token")));

const handleLoggedIn = (token: string) => {
  localStorage.setItem("xhp_token", token);
  authed.value = true;
};

const handleLogout = async () => {
  try {
    await api.logout();
  } catch {
    // token may already be expired
  }
  localStorage.removeItem("xhp_token");
  authed.value = false;
};

const forceLogout = () => {
  authed.value = false;
};

onMounted(async () => {
  window.addEventListener("xhp:logout", forceLogout);
  if (authed.value) {
    try {
      await api.me();
    } catch {
      authed.value = false;
    }
  }
});

onUnmounted(() => window.removeEventListener("xhp:logout", forceLogout));
</script>

