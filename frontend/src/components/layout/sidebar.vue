<script setup lang="ts">
import { ref } from 'vue'

const accountantOpen = ref(true)

const toggleAccountant = () => {
  accountantOpen.value = !accountantOpen.value
}
</script>

<template>
  <aside class="app-sidebar">
    <div class="sidebar-brand">
      <h1 class="brand-logo">RAINBOW</h1>
      <p class="brand-subtitle">CMS</p>
    </div>

    <nav class="sidebar-nav" aria-label="Primary navigation">
      <a href="#" class="nav-item disabled" @click.prevent>
        <span class="material-symbols-outlined">dashboard</span>
        <span class="nav-text">Dashboard</span>
      </a>

      <div class="nav-group" :class="{ 'is-open': accountantOpen }">
        <button class="nav-item group-toggle active" type="button" @click="toggleAccountant">
          <span class="material-symbols-outlined filled">account_balance_wallet</span>
          <span class="nav-text">Accountant</span>
          <span class="material-symbols-outlined arrow">expand_more</span>
        </button>
        <div class="group-items">
          <router-link to="/reserved-fund" class="sub-item" active-class="active">
            <span class="status-indicator-dot"></span>
            <span class="sub-text">Shopping Credits</span>
          </router-link>
        </div>
      </div>

      <a href="#" class="nav-item disabled" @click.prevent>
        <span class="material-symbols-outlined">receipt_long</span>
        <span class="nav-text">Invoices</span>
      </a>

      <a href="#" class="nav-item disabled" @click.prevent>
        <span class="material-symbols-outlined">payments</span>
        <span class="nav-text">Payroll</span>
      </a>

      <a href="#" class="nav-item disabled" @click.prevent>
        <span class="material-symbols-outlined">analytics</span>
        <span class="nav-text">Reports</span>
      </a>

      <a href="#" class="nav-item disabled" @click.prevent>
        <span class="material-symbols-outlined">settings</span>
        <span class="nav-text">Settings</span>
      </a>
    </nav>

    <div class="sidebar-footer">
      <a href="#" class="footer-item">
        <span class="material-symbols-outlined">help</span>
        <span>Help Center</span>
      </a>
      <a href="#" class="footer-item logout-btn">
        <span class="material-symbols-outlined">logout</span>
        <span>Logout</span>
      </a>
    </div>
  </aside>
</template>

<style scoped>
.app-sidebar {
  width: var(--sidebar-width, 260px);
  background: var(--surface);
  border-right: 1px solid var(--outline-variant);
  height: 100vh;
  position: fixed;
  left: 0;
  top: 0;
  display: flex;
  flex-direction: column;
  padding: 1.5rem 1rem;
  z-index: 100;
  color: var(--text-primary);
}

.sidebar-brand {
  padding: 0.5rem 1rem 1.5rem;
  margin-bottom: 1.5rem;
}

.brand-logo {
  color: var(--primary);
  font-size: 1.375rem;
  font-weight: 700;
  line-height: 1.35;
}

.brand-subtitle {
  color: var(--text-muted);
  font-size: 0.75rem;
  font-weight: 600;
  margin-top: 0.25rem;
}

.sidebar-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-weight: 500;
  cursor: pointer;
  transition: var(--transition-fast);
  border: none;
  background: transparent;
  width: 100%;
  text-align: left;
  font-family: inherit;
  font-size: 0.95rem;
}

.nav-item:hover:not(.disabled) {
  background: var(--surface-container-high);
  color: var(--primary);
}

.nav-item.disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.nav-item.active {
  background: var(--secondary-fixed);
  color: var(--on-secondary-fixed);
  border-left: 4px solid var(--primary);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
}

.nav-item span {
  font-size: 1.25rem;
}

.nav-item .filled {
  font-variation-settings: 'FILL' 1;
}

.group-toggle {
  position: relative;
}

.arrow {
  position: absolute;
  right: 1rem;
  transition: transform 0.2s ease;
}

.nav-group.is-open .arrow {
  transform: rotate(180deg);
}

.group-items {
  max-height: 0;
  overflow: hidden;
  transition: max-height 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  flex-direction: column;
  padding-left: 1rem;
  margin: 0.1rem 0 0.5rem 2rem;
  border-left: 1px solid rgba(196, 198, 212, 0.5);
}

.nav-group.is-open .group-items {
  max-height: 200px;
}

.sub-item {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.6rem 1rem;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  font-size: 0.85rem;
  font-weight: 500;
  transition: var(--transition-fast);
}

.sub-item:hover {
  color: var(--primary);
  background: rgba(0, 36, 106, 0.04);
}

.sub-item.active {
  color: var(--primary);
  font-weight: 600;
}

.status-indicator-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-muted);
}

.sub-item.active .status-indicator-dot {
  background: var(--primary);
}

.sidebar-footer {
  border-top: 1px solid rgba(196, 198, 212, 0.5);
  padding-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.footer-item {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 0.6rem 1rem;
  color: var(--text-muted);
  font-size: 0.875rem;
  border-radius: var(--radius-sm);
}

.footer-item:hover {
  background: var(--surface-container-high);
  color: var(--primary);
}

.logout-btn:hover {
  color: var(--danger);
}

@media (max-width: 720px) {
  .app-sidebar {
    display: none;
  }
}
</style>
