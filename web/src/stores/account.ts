import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { accountApi, type Account } from '@/api'

export const useAccountStore = defineStore('account', () => {
  const accounts = ref<Account[]>([])
  const selectedAccountId = ref<number | null>(null)
  const isLoading = ref(false)

  const selectedAccount = computed<Account | null>(() =>
    accounts.value.find(a => a.id === selectedAccountId.value) ?? null
  )

  async function fetchAccounts(): Promise<void> {
    isLoading.value = true
    try {
      const res = await accountApi.getAll()
      accounts.value = res.data
    } catch {
      // Silent fail - accounts will be empty
    } finally {
      isLoading.value = false
    }
  }

  function selectAccount(id: number | null): void {
    selectedAccountId.value = id
    if (id !== null) {
      // Persist to localStorage
      localStorage.setItem('selectedAccountId', String(id))
    } else {
      localStorage.removeItem('selectedAccountId')
    }
  }

  function loadPersistedAccount(): void {
    const persistedId = localStorage.getItem('selectedAccountId')
    if (persistedId) {
      const id = parseInt(persistedId, 10)
      if (!isNaN(id)) {
        selectedAccountId.value = id
      }
    }
  }

  return {
    accounts,
    selectedAccountId,
    selectedAccount,
    isLoading,
    fetchAccounts,
    selectAccount,
    loadPersistedAccount
  }
})
