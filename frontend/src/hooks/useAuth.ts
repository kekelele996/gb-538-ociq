import { computed } from 'vue'
import { useAuthStore } from '../stores/auth-store'
import { can } from '../utils/permissions'

export function useAuth() {
  const auth = useAuthStore()
  return {
    auth,
    canWritePoints: computed(() => can(auth.user?.role, 'pointWrite')),
    canImport: computed(() => can(auth.user?.role, 'measurementImport')),
    canFlowMeasurement: computed(() => can(auth.user?.role, 'measurementFlow')),
    canWriteSources: computed(() => can(auth.user?.role, 'sourceWrite')),
    canRun: computed(() => can(auth.user?.role, 'run')),
    canReview: computed(() => can(auth.user?.role, 'review')),
    canConfirm: computed(() => can(auth.user?.role, 'confirm')),
    canAudit: computed(() => can(auth.user?.role, 'audit')),
  }
}
