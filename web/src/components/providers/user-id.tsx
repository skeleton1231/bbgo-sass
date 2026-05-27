'use client'

import { createContext, useContext } from 'react'

const UserIdContext = createContext<string>('')

export function UserIdProvider({ userId, children }: { userId: string; children: React.ReactNode }) {
  return (
    <UserIdContext.Provider value={userId}>{children}</UserIdContext.Provider>
  )
}

export function useUserId() {
  return useContext(UserIdContext)
}
