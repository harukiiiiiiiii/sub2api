<template>
  <button type="button" class="btn btn-secondary text-xs" @click="open">
    {{ t('admin.groups.quotaReset.action') }}
  </button>
  <BaseDialog :show="show" :title="t('admin.groups.quotaReset.title')" width="narrow" @close="close">
    <div class="space-y-4">
      <p class="text-sm text-gray-700 dark:text-gray-200">
        {{ t('admin.groups.quotaReset.scope', { name: group.name }) }}
      </p>
      <p class="text-sm text-gray-500">
        {{ t('admin.groups.quotaReset.hint') }}
      </p>
      <p v-if="loading" role="status">{{ t('common.loading') }}</p>
      <p v-else-if="count !== null" class="text-sm">
        {{ t('admin.groups.quotaReset.count', { count }) }}
      </p>
      <fieldset :disabled="busy" class="flex flex-wrap gap-4">
        <legend class="mb-2 text-sm font-medium">{{ t('admin.groups.quotaReset.windows') }}</legend>
        <label v-for="window in windows" :key="window" class="flex items-center gap-2 text-sm">
          <input v-model="selection[window]" type="checkbox" :name="window" class="rounded" @change="operationKey = ''" />
          {{ t(`admin.groups.quotaReset.${window}`) }}
        </label>
      </fieldset>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
    </div>
    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="busy" @click="close">{{ t('common.cancel') }}</button>
      <button type="button" class="btn btn-primary" :disabled="busy || loading || !count || !hasSelection" @click="reset">
        {{ busy ? t('common.processing') : t('admin.groups.quotaReset.confirm') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { listByGroup, resetGroupQuota } from '@/api/admin/subscriptions'
import { useAppStore } from '@/stores/app'

const props = defineProps<{ group: { id: number; name: string } }>()
const emit = defineEmits<{ (e: 'reset'): void }>()
const { t } = useI18n()
const app = useAppStore()
const windows = ['daily', 'weekly', 'monthly'] as const
const selection = reactive({ daily: false, weekly: false, monthly: false })
const hasSelection = computed(() => windows.some(window => selection[window]))
const show = ref(false)
const busy = ref(false)
const loading = ref(false)
const count = ref<number | null>(null)
const error = ref('')
// Retain the operation key after ambiguous failures so retrying cannot clear
// newly accrued usage a second time.
const operationKey = ref('')

function errorMessage(err: unknown): string {
  const candidate = err as { response?: { data?: { message?: string } }; message?: string }
  return candidate?.response?.data?.message || candidate?.message || t('admin.groups.quotaReset.failed')
}

async function open() {
  show.value = true
  loading.value = true
  error.value = ''
  count.value = null
  try {
    const result = await listByGroup(props.group.id, 1, 1)
    count.value = result.total
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    loading.value = false
  }
}

function close() {
  if (!busy.value) show.value = false
}

async function reset() {
  if (busy.value || loading.value || !count.value || !hasSelection.value) return
  busy.value = true
  error.value = ''
  if (!operationKey.value) {
    const id = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
    operationKey.value = `group-quota-${props.group.id}-${id}`
  }
  try {
    const result = await resetGroupQuota(props.group.id, { ...selection }, operationKey.value)
    app.showSuccess(t('admin.groups.quotaReset.success', { count: result.reset_count }))
    if (result.cache_warnings) app.showWarning(t('admin.groups.quotaReset.cacheWarning'))
    operationKey.value = ''
    windows.forEach(window => { selection[window] = false })
    show.value = false
    emit('reset')
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    busy.value = false
  }
}
</script>
