import { SignInForm } from '@/components/auth/SignInForm'

export default function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="w-full max-w-sm space-y-6 p-6">
        <h1 className="text-2xl font-bold text-center">BBGO</h1>
        <SignInForm />
      </div>
    </div>
  )
}
