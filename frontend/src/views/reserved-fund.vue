<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const isDragging = ref(false)
const selectedFile = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const isProcessing = ref(false)

const acceptedExtensions = ['.xlsx']

const fileSizeLabel = computed(() => {
  if (!selectedFile.value) return ''

  const size = selectedFile.value.size
  if (size >= 1024 * 1024) {
    return `${(size / (1024 * 1024)).toFixed(1)} MB`
  }

  return `${(size / 1024).toFixed(1)} KB`
})

const onDragOver = (e: DragEvent) => {
  e.preventDefault()
  isDragging.value = true
}

const onDragLeave = () => {
  isDragging.value = false
}

const onDrop = (e: DragEvent) => {
  e.preventDefault()
  isDragging.value = false

  const file = e.dataTransfer?.files[0]
  if (file) validateAndSetFile(file)
}

const onFileChange = (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (file) validateAndSetFile(file)
}

const validateAndSetFile = (file: File) => {
  const ext = `.${file.name.split('.').pop()?.toLowerCase()}`
  if (!acceptedExtensions.includes(ext)) {
    alert('Unsupported file type. Please select .xlsx')
    return
  }

  selectedFile.value = file
}

const openFilePicker = () => {
  fileInput.value?.click()
}

const handleImportRequest = () => {
  openFilePicker()
}

const removeFile = () => {
  selectedFile.value = null
  if (fileInput.value) fileInput.value.value = ''
}

const processError = ref<string | null>(null)
const processSuccess = ref<string | null>(null)

const process = async () => {
  if (!selectedFile.value) return

  processError.value = null
  processSuccess.value = null
  isProcessing.value = true

  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)

    const res = await fetch('/api/group-reserved-funds/import-470', {
      method: 'POST',
      body: formData,
    })

    const data = await res.json()

    if (!res.ok) {
      processError.value = data?.error ?? `Server error (${res.status})`
    } else {
      processSuccess.value = `File "${selectedFile.value.name}" submitted — batch #${data.id} is being processed.`
      removeFile()
    }
  } catch (err) {
    processError.value = 'Network error. Please try again.'
  } finally {
    isProcessing.value = false
  }
}

onMounted(() => {
  window.addEventListener('open-reserved-fund-import', handleImportRequest)
})

onBeforeUnmount(() => {
  window.removeEventListener('open-reserved-fund-import', handleImportRequest)
})

// ── Export section state ──────────────────────────────────────────────────────
const isDraggingEx = ref(false)
const selectedFileEx = ref<File | null>(null)
const fileInputEx = ref<HTMLInputElement | null>(null)
const isExporting = ref(false)
const exportError = ref<string | null>(null)
const exportSuccess = ref<string | null>(null)
const exportBatchId = ref<number | null>(null)

const fileSizeLabelEx = computed(() => {
  if (!selectedFileEx.value) return ''
  const size = selectedFileEx.value.size
  if (size >= 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(1)} MB`
  return `${(size / 1024).toFixed(1)} KB`
})

const onDragOverEx = (e: DragEvent) => {
  e.preventDefault()
  isDraggingEx.value = true
}

const onDragLeaveEx = () => {
  isDraggingEx.value = false
}

const onDropEx = (e: DragEvent) => {
  e.preventDefault()
  isDraggingEx.value = false
  const file = e.dataTransfer?.files[0]
  if (file) validateAndSetFileEx(file)
}

const onFileChangeEx = (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (file) validateAndSetFileEx(file)
}

const validateAndSetFileEx = (file: File) => {
  const ext = `.${file.name.split('.').pop()?.toLowerCase()}`
  if (!acceptedExtensions.includes(ext)) {
    alert('Unsupported file type. Please select .xlsx, .csv, or .json')
    return
  }
  selectedFileEx.value = file
}

const openFilePickerEx = () => {
  fileInputEx.value?.click()
}

const removeFileEx = () => {
  selectedFileEx.value = null
  if (fileInputEx.value) fileInputEx.value.value = ''
}

const exportFile = async () => {
  if (!selectedFileEx.value) return

  exportError.value = null
  exportSuccess.value = null
  isExporting.value = true

  try {
    const formData = new FormData()
    formData.append('file', selectedFileEx.value)

    const res = await fetch('/api/reserved-fund-usage/export', {
      method: 'POST',
      body: formData,
    })

    const data = await res.json().catch(() => ({}))

    if (!res.ok) {
      exportError.value = data?.error ?? `Server error (${res.status})`
      return
    }

    // Backend accepted the file and is processing asynchronously.
    // The result will be saved on the server; link to the download endpoint.
    exportBatchId.value = data.id ?? null
    exportSuccess.value = `File "${selectedFileEx.value!.name}" submitted — batch #${data.id} is being processed.`
    removeFileEx()
  } catch (err) {
    exportError.value = 'Network error. Please try again.'
  } finally {
    isExporting.value = false
  }
}
</script>

<template>
  <section class="rf-page" aria-label="Shopping credits import">
    <div class="import-stack">

      <div class="header">
        <h1>470 Import</h1>
        <p>Upload a "470" file to import shopping credits.</p>
      </div>
      <div
        class="drop-zone"
        :class="{ dragging: isDragging, 'has-file': selectedFile }"
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onDrop"
        @click="!selectedFile && openFilePicker()"
      >
        <input
          ref="fileInput"
          type="file"
          accept=".xlsx,.csv,.json"
          class="hidden-input"
          @change="onFileChange"
        />

        <template v-if="!selectedFile">
          <div class="drop-icon">
            <span class="material-symbols-outlined">cloud_upload</span>
          </div>
          <h3 class="drop-title">Click or drag .xlsx files here to import</h3>
          <p class="drop-subtitle">Supports .xlsx only</p>
          <button class="btn-select" type="button" @click.stop="openFilePicker">
            Select File
          </button>
        </template>

        <template v-else>
          <div class="file-preview">
            <div class="file-icon">
              <span class="material-symbols-outlined">description</span>
            </div>
            <div class="file-info">
              <p class="file-status">Ready to process</p>
              <p class="file-name">{{ selectedFile.name }}</p>
              <p class="file-size">{{ fileSizeLabel }}</p>
            </div>
            <button class="btn-remove" type="button" title="Remove file" @click.stop="removeFile">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>
        </template>
      </div>

      <p v-if="processError" class="feedback feedback--error">
        <span class="material-symbols-outlined">error</span>
        {{ processError }}
      </p>
      <p v-if="processSuccess" class="feedback feedback--success">
        <span class="material-symbols-outlined">check_circle</span>
        {{ processSuccess }}
      </p>

      <div class="actions-row">
        <button
          id="btn-process-reserved-fund"
          class="btn-process"
          :disabled="!selectedFile || isProcessing"
          type="button"
          @click="process"
        >
          <span v-if="isProcessing" class="material-symbols-outlined spinner">progress_activity</span>
          <span>{{ isProcessing ? 'Processing...' : 'Process' }}</span>
        </button>
      </div>
    </div>
  </section>

  <section class="rf-page" aria-label="Reserved fund usage export">
    <div class="import-stack">
      <div class="header">
        <h1>购物金 Export</h1>
        <p>Export the reserved fund usage to an Excel file.</p>
      </div>
      <div
        class="drop-zone"
        :class="{ dragging: isDraggingEx, 'has-file': selectedFileEx }"
        @dragover="onDragOverEx"
        @dragleave="onDragLeaveEx"
        @drop="onDropEx"
        @click="!selectedFileEx && openFilePickerEx()"
      >
        <input
          ref="fileInputEx"
          type="file"
          accept=".xlsx,.csv,.json"
          class="hidden-input"
          @change="onFileChangeEx"
        />

        <template v-if="!selectedFileEx">
          <div class="drop-icon">
            <span class="material-symbols-outlined">cloud_download</span>
          </div>
          <h3 class="drop-title">Click or drag files here to export</h3>
          <p class="drop-subtitle">Supports .xlsx only</p>
          <button class="btn-select" type="button" @click.stop="openFilePickerEx">
            Select File
          </button>
        </template>

        <template v-else>
          <div class="file-preview">
            <div class="file-icon">
              <span class="material-symbols-outlined">description</span>
            </div>
            <div class="file-info">
              <p class="file-status">Ready to export</p>
              <p class="file-name">{{ selectedFileEx.name }}</p>
              <p class="file-size">{{ fileSizeLabelEx }}</p>
            </div>
            <button class="btn-remove" type="button" title="Remove file" @click.stop="removeFileEx">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>
        </template>
      </div>

      <p v-if="exportError" class="feedback feedback--error">
        <span class="material-symbols-outlined">error</span>
        {{ exportError }}
      </p>
      <p v-if="exportSuccess" class="feedback feedback--success">
        <span class="material-symbols-outlined">check_circle</span>
        <span>
          {{ exportSuccess }}
          <a
            v-if="exportBatchId"
            :href="`/api/batches/${exportBatchId}/download/result`"
            class="download-link"
            target="_blank"
          >Download result</a>
        </span>
      </p>

      <div class="actions-row">
        <button
          id="btn-export-reserved-fund"
          class="btn-process"
          :disabled="!selectedFileEx || isExporting"
          type="button"
          @click="exportFile"
        >
          <span v-if="isExporting" class="material-symbols-outlined spinner">progress_activity</span>
          <span>{{ isExporting ? 'Exporting...' : 'Export' }}</span>
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.rf-page {
  min-height: calc(85vh - 88px);
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.import-stack {
  width: 100%;
  max-width: 672px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.drop-zone {
  min-height: 316px;
  border: 2px dashed var(--outline-variant);
  border-radius: 12px;
  background: var(--surface-container-lowest);
  padding: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  text-align: center;
  transition: var(--transition-fast);
}

.drop-zone:hover,
.drop-zone.dragging {
  border-color: var(--primary);
  background: rgba(0, 36, 106, 0.05);
}

.drop-zone.dragging .drop-icon,
.drop-zone:hover .drop-icon {
  transform: scale(1.08);
}

.drop-zone.has-file {
  cursor: default;
  padding: 32px;
}

.hidden-input {
  display: none;
}

.drop-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 24px;
  border-radius: 999px;
  background: rgba(0, 36, 106, 0.1);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.2s ease;
}

.drop-icon .material-symbols-outlined {
  font-size: 40px;
}

.drop-title {
  color: var(--text-primary);
  font-size: 1.375rem;
  line-height: 1.36;
  font-weight: 600;
  margin-bottom: 8px;
}

.drop-subtitle {
  color: var(--text-secondary);
  font-size: 0.875rem;
  line-height: 1.45;
}

.btn-select,
.btn-process {
  border: none;
  background: var(--primary);
  color: var(--on-primary);
  border-radius: var(--radius-sm);
  min-height: 44px;
  padding: 0 24px;
  font-family: inherit;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  box-shadow: var(--shadow-sm);
  transition: var(--transition-fast);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  white-space: nowrap;
}

.btn-select {
  margin-top: 32px;
}

.btn-select:hover,
.btn-process:hover:not(:disabled) {
  background: var(--primary-container);
}

.file-preview {
  width: 100%;
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr) 36px;
  align-items: center;
  gap: 16px;
  border: 1px solid var(--outline-variant);
  border-radius: var(--radius-md);
  background: var(--surface-container-low);
  padding: 16px 20px;
  text-align: left;
}

.file-icon {
  width: 52px;
  height: 52px;
  border-radius: var(--radius-sm);
  background: var(--secondary-fixed);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-icon .material-symbols-outlined {
  font-size: 26px;
}

.file-info {
  min-width: 0;
}

.file-status {
  color: var(--success);
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.35;
}

.file-name {
  color: var(--text-primary);
  font-size: 0.95rem;
  font-weight: 600;
  line-height: 1.45;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  color: var(--text-muted);
  font-size: 0.8rem;
  line-height: 1.35;
}

.btn-remove {
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--transition-fast);
}

.btn-remove:hover {
  background: var(--error-container, #ffdad6);
  color: var(--danger);
}

.btn-remove .material-symbols-outlined {
  font-size: 20px;
}

.actions-row {
  display: flex;
  justify-content: flex-end;
}

.btn-process {
  min-width: 112px;
}

.btn-process:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.spinner {
  font-size: 18px;
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.feedback {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: var(--radius-sm);
  font-size: 0.875rem;
  font-weight: 500;
  line-height: 1.45;
}

.feedback .material-symbols-outlined {
  font-size: 18px;
  flex-shrink: 0;
}

.feedback--error {
  background: var(--error-container, #ffdad6);
  color: var(--danger, #b3261e);
}

.feedback--success {
  background: rgba(0, 128, 0, 0.1);
  color: var(--success, #2e7d32);
}

.download-link {
  margin-left: 8px;
  color: var(--primary);
  font-weight: 600;
  text-decoration: underline;
  white-space: nowrap;
}

@media (max-width: 640px) {
  .rf-page {
    min-height: calc(100vh - 96px);
    align-items: flex-start;
  }

  .drop-zone {
    min-height: 280px;
    padding: 32px 20px;
  }

  .file-preview {
    grid-template-columns: 44px minmax(0, 1fr) 36px;
    padding: 14px;
  }

  .file-icon {
    width: 44px;
    height: 44px;
  }
}
</style>
