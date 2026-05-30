import type { NextConfig } from 'next'
import createNextIntlPlugin from 'next-intl/plugin'

const withNextIntl = createNextIntlPlugin({
  experimental: {
    createMessagesDeclaration: './messages/en.json',
  },
  requestConfig: './src/i18n/request.ts',
})

const nextConfig: NextConfig = {
  turbopack: {},
  experimental: {
    staleTimes: {
      dynamic: 30,
      static: 180,
    },
    optimizePackageImports: [
      '@supabase/supabase-js',
      'lucide-react',
      '@tanstack/react-query',
      'recharts',
      'next-intl',
      'next-themes',
    ],
  },
}

export default withNextIntl(nextConfig)
