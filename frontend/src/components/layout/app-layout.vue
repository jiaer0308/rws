<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Sidebar from './sidebar.vue'

interface HealthStatus {
  status: string
  database: string
  time: string
}

const loading = ref(false)
const error = ref<string | null>(null)
const data = ref<HealthStatus | null>(null)

const checkHealth = async () => {
  loading.value = true
  error.value = null
  try {
    const res = await fetch('/api/health')
    if (!res.ok) {
      throw new Error(`HTTP error! status: ${res.status}`)
    }
    data.value = await res.json()
  } catch (err: any) {
    console.error(err)
    error.value = err.message || 'Offline'
  } finally {
    loading.value = false
  }
}

const requestImport = () => {
  window.dispatchEvent(new CustomEvent('open-reserved-fund-import'))
}

onMounted(() => {
  checkHealth()
})
</script>

<template>
  <div class="app-layout">
    <Sidebar />

    <div class="main-viewport">
      <header class="top-nav">
        <div class="nav-left">
          <h2 class="nav-title">Accountant Dashboard</h2>
        </div>

        <div class="nav-right">
          <!-- <button class="icon-button" title="Notifications">
            <span class="material-symbols-outlined">notifications</span>
          </button>
          <button class="icon-button" title="Messages">
            <span class="material-symbols-outlined">chat_bubble</span>
          </button>

          <button class="import-button" type="button" @click="requestImport">
            <span class="material-symbols-outlined">upload_file</span>
            <span>Import Excel</span>
          </button> -->

          <!-- <button
            class="nav-status-badge"
            :class="{ 'status-ok': data && data.status === 'healthy', 'status-err': error }"
            type="button"
            title="Refresh connection status"
            @click="checkHealth"
          >
            <span class="status-dot"></span>
            {{ error ? 'Offline' : (loading ? 'Connecting' : 'Online') }}
          </button> -->

          <div class="user-profile">
            <div class="profile-info">
              <span class="user-name">Accountant</span>
              <span class="user-role">Finance Lead</span>
            </div>
            <div class="avatar-wrapper">
              <img 
                alt="Accountant Profile" 
                class="avatar-image" 
                src="https://lh3.googleusercontent.com/aida-public/AB6AXuCEjTyanXrmDAFkPUOHKmiAZCCUp6pBHdKx9cukowhf5Oj6Zp9Td0m_4jmVTiWzYziB4RKAVIOggPnkXVJ0xkukfi-bfl0bb-qfXkr1Q6uXHZmTHFwX7kj3jy2ZxVwcaR0ufMY2V1hoJ2JLrANLPbTJYuZ8xsmGCmv16d76SkFuebdzR5mvxCyPSr6OWj8qqzXp7ro_iFvDXP0MZuo8cNCKRvbLpsq2-PKsJEMed3rYcKQxkdxGR7je7qP_FXIzndGuVw3udrsZ-G6s"
              >
            </div>
          </div>
        </div>
      </header>

      <main class="page-container">
        <slot></slot>
      </main>
    </div>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  min-height: 100vh;
  background: var(--bg-main);
  color: var(--text-primary);
}

.main-viewport {
  flex: 1;
  margin-left: var(--sidebar-width, 260px);
  display: flex;
  flex-direction: column;
  min-width: 0; /* Prevents flex items from overflowing */
}

.top-nav {
  height: 64px;
  background: rgba(250, 251, 251, 0.88);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(196, 198, 212, 0.75);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  position: sticky;
  top: 0;
  z-index: 90;
  box-shadow: 0 1px 3px rgba(6, 7, 11, 0.04);
}

.nav-left {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  min-width: 0;
}

.nav-title {
  color: var(--primary);
  font-size: 1.125rem;
  font-weight: 700;
  line-height: 1.45;
  white-space: nowrap;
}

.nav-tabs {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.nav-tab {
  color: var(--text-secondary);
  font-size: 0.875rem;
  font-weight: 500;
  line-height: 1.45;
  padding: 0.25rem 0;
  border-bottom: 2px solid transparent;
}

.nav-tab:hover {
  color: var(--primary);
}

.nav-tab.active {
  color: var(--primary);
  border-bottom-color: var(--primary);
  font-weight: 700;
}

.nav-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.nav-status-badge {
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-lowest);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1;
  padding: 0.55rem 0.75rem;
  border-radius: 999px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: var(--transition-fast);
  font-family: inherit;
}

.nav-status-badge:hover {
  border-color: rgba(0, 36, 106, 0.2);
  color: var(--primary);
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-muted);
}

.status-ok .status-dot {
  background: var(--success);
}

.status-err .status-dot {
  background: var(--danger);
}

.icon-button {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  transition: var(--transition-fast);
}

.icon-button:hover {
  background: var(--surface-container-lowest);
  color: var(--primary);
}

.import-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  border: none;
  background: var(--primary);
  color: var(--on-primary);
  border-radius: var(--radius-sm);
  padding: 0.6rem 1rem;
  font: inherit;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  box-shadow: var(--shadow-sm);
  transition: var(--transition-fast);
  white-space: nowrap;
}

.import-button:hover {
  background: var(--primary-container);
}

.import-button .material-symbols-outlined {
  font-size: 20px;
}

.user-profile {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.profile-info {
  display: flex;
  flex-direction: column;
  text-align: right;
}

.user-name {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1.2;
}

.user-role {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.avatar-wrapper {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  overflow: hidden;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-high);
}

.avatar-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.page-container {
  flex: 1;
  padding: 24px 24px 32px;
  max-width: 1440px;
  width: 100%;
  margin: 0 auto;
}

@media (max-width: 960px) {
  .nav-tabs,
  .profile-info,
  .nav-status-badge {
    display: none;
  }
}

@media (max-width: 720px) {
  .main-viewport {
    margin-left: 0;
  }

  .top-nav {
    padding-left: 16px;
    padding-right: 16px;
  }

  .import-button span:last-child {
    display: none;
  }
}
</style>
