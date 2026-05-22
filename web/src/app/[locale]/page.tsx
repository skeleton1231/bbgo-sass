import { redirect } from 'next/navigation'
import { USER_DASHBOARD_PATH } from '@/lib/routes'

export default function LocaleRoot() {
  redirect(USER_DASHBOARD_PATH)
}
