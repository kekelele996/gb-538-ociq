import type { UserRole } from '../types/auth'

type Capability = 'pointWrite' | 'measurementImport' | 'measurementFlow' | 'sourceWrite' | 'run' | 'review' | 'confirm' | 'audit'

const permissions: Record<UserRole, Capability[]> = {
  admin: ['pointWrite', 'measurementImport', 'measurementFlow', 'sourceWrite', 'run', 'review', 'confirm', 'audit'],
  acoustic_engineer: ['pointWrite', 'measurementImport', 'measurementFlow', 'sourceWrite', 'run'],
  data_analyst: ['measurementImport', 'measurementFlow', 'run'],
  reviewer: ['review', 'confirm', 'audit'],
  auditor: ['audit'],
}

export const can = (role: UserRole | undefined, capability: Capability) => Boolean(role && permissions[role].includes(capability))
