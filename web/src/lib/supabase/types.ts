export type Json =
  | string
  | number
  | boolean
  | null
  | { [key: string]: Json | undefined }
  | Json[]

export interface Database {
  public: {
    Tables: {
      user_profiles: {
        Row: {
          id: string
          email: string
          display_name: string | null
          role: string
          avatar_url: string | null
          created_at: string
          updated_at: string
        }
        Insert: {
          id: string
          email: string
          display_name?: string | null
          role?: string
          avatar_url?: string | null
          created_at?: string
          updated_at?: string
        }
        Update: {
          id?: string
          email?: string
          display_name?: string | null
          role?: string
          avatar_url?: string | null
          created_at?: string
          updated_at?: string
        }
      }
      bots: {
        Row: {
          id: string
          user_id: string
          name: string
          exchange: string
          strategy: string
          config: Json
          mode: 'live' | 'paper'
          status: 'running' | 'stopped' | 'error'
          bbgo_pid: number | null
          webserver_port: number | null
          grpc_port: number | null
          config_path: string | null
          created_at: string
          updated_at: string
        }
        Insert: {
          id?: string
          user_id: string
          name: string
          exchange: string
          strategy: string
          config: Json
          mode?: 'live' | 'paper'
          status?: 'stopped'
          bbgo_pid?: number | null
          webserver_port?: number | null
          grpc_port?: number | null
          config_path?: string | null
          created_at?: string
          updated_at?: string
        }
        Update: {
          id?: string
          name?: string
          config?: Json
          mode?: 'live' | 'paper'
          status?: 'running' | 'stopped' | 'error'
          bbgo_pid?: number | null
          webserver_port?: number | null
          grpc_port?: number | null
          config_path?: string | null
          updated_at?: string
        }
      }
      exchange_credentials: {
        Row: {
          id: string
          user_id: string
          exchange: string
          api_key_encrypted: string
          api_secret_encrypted: string
          passphrase_encrypted: string | null
          is_testnet: boolean
          is_verified: boolean
          last_verified_at: string | null
          created_at: string
        }
        Insert: {
          id?: string
          user_id: string
          exchange: string
          api_key_encrypted: string
          api_secret_encrypted: string
          passphrase_encrypted?: string | null
          is_testnet?: boolean
          is_verified?: boolean
          last_verified_at?: string | null
          created_at?: string
        }
        Update: {
          api_key_encrypted?: string
          api_secret_encrypted?: string
          passphrase_encrypted?: string | null
          is_testnet?: boolean
          is_verified?: boolean
          last_verified_at?: string | null
        }
      }
      sync_orders: {
        Row: {
          id: string
          bot_id: string
          user_id: string
          symbol: string
          side: string
          type: string
          price: string
          quantity: string
          status: string
          order_id: string
          synced_at: string
          created_at: string
        }
        Insert: {
          id?: string
          bot_id: string
          user_id: string
          symbol: string
          side: string
          type: string
          price: string
          quantity: string
          status: string
          order_id: string
          synced_at?: string
          created_at?: string
        }
        Update: {
          status?: string
          synced_at?: string
        }
      }
      sync_trades: {
        Row: {
          id: string
          bot_id: string
          user_id: string
          symbol: string
          side: string
          price: string
          quantity: string
          fee: string
          fee_currency: string
          trade_id: string
          order_id: string
          pnl: string | null
          synced_at: string
          created_at: string
        }
        Insert: {
          id?: string
          bot_id: string
          user_id: string
          symbol: string
          side: string
          price: string
          quantity: string
          fee: string
          fee_currency: string
          trade_id: string
          order_id: string
          pnl?: string | null
          synced_at?: string
          created_at?: string
        }
        Update: {
          synced_at?: string
        }
      }
      sync_cursors: {
        Row: {
          user_id: string
          table_name: string
          last_gid: number
          updated_at: string
        }
        Insert: {
          user_id: string
          table_name: string
          last_gid?: number
          updated_at?: string
        }
        Update: {
          last_gid?: number
          updated_at?: string
        }
      }
      backtest_reports: {
        Row: {
          id: string
          user_id: string
          strategy: string
          config: Json
          start_date: string
          end_date: string
          total_profit: string
          max_drawdown: string
          sharpe_ratio: string | null
          sortino_ratio: string | null
          profit_factor: string | null
          win_rate: string
          total_trades: number
          win_count: number
          loss_count: number
          cagr: string | null
          report_json: Json
          created_at: string
        }
        Insert: {
          id?: string
          user_id: string
          strategy: string
          config: Json
          start_date: string
          end_date: string
          total_profit: string
          max_drawdown: string
          sharpe_ratio?: string | null
          sortino_ratio?: string | null
          profit_factor?: string | null
          win_rate: string
          total_trades: number
          win_count: number
          loss_count: number
          cagr?: string | null
          report_json: Json
          created_at?: string
        }
        Update: {
          report_json?: Json
        }
      }
    }
    Views: {}
    Functions: {}
    Enums: {}
  }
}
