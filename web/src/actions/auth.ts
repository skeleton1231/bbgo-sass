'use server'

import { revalidatePath } from 'next/cache'
import { redirect } from 'next/navigation'
import { createClient } from '@/lib/supabase/server'
import { LOGIN_PATH, USER_PATH } from '@/lib/routes'

export async function signInAction(prevState: unknown, formData: FormData) {
  const email = formData.get('email') as string
  const password = formData.get('password') as string

  if (!email || !password) {
    return { error: 'Email and password are required' }
  }

  try {
    const supabase = await createClient()
    const { error } = await supabase.auth.signInWithPassword({ email, password })

    if (error) {
      return { error: error.message }
    }
  } catch {
    return { error: 'Unable to connect to authentication service. Please try again.' }
  }

  revalidatePath('/')
  redirect(USER_PATH)
}

export async function signOutAction() {
  const supabase = await createClient()
  await supabase.auth.signOut()
  revalidatePath('/')
  redirect(LOGIN_PATH)
}

export async function signUpAction(prevState: unknown, formData: FormData) {
  const email = formData.get('email') as string
  const password = formData.get('password') as string
  const confirmPassword = formData.get('confirmPassword') as string

  if (!email || !password || !confirmPassword) {
    return { error: 'All fields are required' }
  }

  if (password !== confirmPassword) {
    return { error: 'Passwords do not match' }
  }

  try {
    const supabase = await createClient()
    const { error } = await supabase.auth.signUp({ email, password })

    if (error) {
      return { error: error.message }
    }
  } catch {
    return { error: 'Unable to connect to authentication service. Please try again.' }
  }

  revalidatePath('/')
  redirect(LOGIN_PATH + '?signup=1')
}

export async function resetPasswordAction(prevState: unknown, formData: FormData) {
  const email = formData.get('email') as string

  if (!email) {
    return { error: 'Email is required' }
  }

  try {
    const supabase = await createClient()
    const { error } = await supabase.auth.resetPasswordForEmail(email, {
      redirectTo: `${process.env.NEXT_PUBLIC_SITE_URL ?? 'http://localhost:3142'}/auth/confirm`,
    })

    if (error) {
      return { error: error.message }
    }
  } catch {
    return { error: 'Unable to connect to authentication service. Please try again.' }
  }

  return { success: 'Password reset link sent to your email.' }
}
