import { computed } from 'vue'
import { useAttributionRunStore } from '../stores/attribution-run-store'
import { useAuthStore } from '../stores/auth-store'

export function useAttributionRun() {
  const store = useAttributionRunStore()
  const auth = useAuthStore()
  const canReviewSelected = computed(() => store.selected?.attribution_state === 'completed' && ['admin', 'reviewer'].includes(auth.user?.role ?? ''))
  const canConfirmSelected = computed(() => store.selected?.attribution_state === 'reviewed' && ['admin', 'reviewer'].includes(auth.user?.role ?? '') && store.selected?.created_by !== auth.user?.id)
  return { store, canReviewSelected, canConfirmSelected }
}
