export type Json =
  | string
  | number
  | boolean
  | null
  | { [key: string]: Json | undefined }
  | Json[]

export type Database = {
  // Allows to automatically instantiate createClient with right options
  // instead of createClient<Database, { PostgrestVersion: 'XX' }>(URL, KEY)
  __InternalSupabase: {
    PostgrestVersion: "14.5"
  }
  public: {
    Tables: {
      backtest_reports: {
        Row: {
          cagr: string | null
          config: Json
          created_at: string
          end_date: string
          id: string
          loss_count: number
          max_drawdown: string
          profit_factor: string | null
          report_json: Json
          sharpe_ratio: string | null
          sortino_ratio: string | null
          start_date: string
          strategy: string
          total_profit: string
          total_trades: number
          user_id: string
          win_count: number
          win_rate: string
        }
        Insert: {
          cagr?: string | null
          config?: Json
          created_at?: string
          end_date: string
          id?: string
          loss_count: number
          max_drawdown: string
          profit_factor?: string | null
          report_json?: Json
          sharpe_ratio?: string | null
          sortino_ratio?: string | null
          start_date: string
          strategy: string
          total_profit: string
          total_trades: number
          user_id: string
          win_count: number
          win_rate: string
        }
        Update: {
          cagr?: string | null
          config?: Json
          created_at?: string
          end_date?: string
          id?: string
          loss_count?: number
          max_drawdown?: string
          profit_factor?: string | null
          report_json?: Json
          sharpe_ratio?: string | null
          sortino_ratio?: string | null
          start_date?: string
          strategy?: string
          total_profit?: string
          total_trades?: number
          user_id?: string
          win_count?: number
          win_rate?: string
        }
        Relationships: [
          {
            foreignKeyName: "backtest_reports_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      deposits: {
        Row: {
          address: string
          amount: string
          asset: string
          exchange: string
          id: string
          time: string
          txn_id: string
          user_id: string
        }
        Insert: {
          address?: string
          amount?: string
          asset?: string
          exchange?: string
          id?: string
          time?: string
          txn_id?: string
          user_id: string
        }
        Update: {
          address?: string
          amount?: string
          asset?: string
          exchange?: string
          id?: string
          time?: string
          txn_id?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "deposits_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      exchange_credentials: {
        Row: {
          api_key_encrypted: string
          api_secret_encrypted: string
          created_at: string
          exchange: string
          id: string
          is_testnet: boolean
          is_verified: boolean
          last_verified_at: string | null
          passphrase_encrypted: string | null
          user_id: string
        }
        Insert: {
          api_key_encrypted: string
          api_secret_encrypted: string
          created_at?: string
          exchange: string
          id?: string
          is_testnet?: boolean
          is_verified?: boolean
          last_verified_at?: string | null
          passphrase_encrypted?: string | null
          user_id: string
        }
        Update: {
          api_key_encrypted?: string
          api_secret_encrypted?: string
          created_at?: string
          exchange?: string
          id?: string
          is_testnet?: boolean
          is_verified?: boolean
          last_verified_at?: string | null
          passphrase_encrypted?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "exchange_credentials_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      futures_position_risks: {
        Row: {
          adl: string
          break_even_price: string
          entry_price: string
          exchange: string
          id: string
          initial_margin: string
          leverage: string
          liquidation_price: string
          maint_margin: string
          margin_asset: string
          mark_price: string
          notional: string
          open_order_initial_margin: string
          position_amount: string
          position_initial_margin: string
          position_side: string
          symbol: string
          unrealized_pnl: string
          updated_at: string
          user_id: string
        }
        Insert: {
          adl?: string
          break_even_price?: string
          entry_price?: string
          exchange?: string
          id?: string
          initial_margin?: string
          leverage?: string
          liquidation_price?: string
          maint_margin?: string
          margin_asset?: string
          mark_price?: string
          notional?: string
          open_order_initial_margin?: string
          position_amount?: string
          position_initial_margin?: string
          position_side?: string
          symbol?: string
          unrealized_pnl?: string
          updated_at?: string
          user_id: string
        }
        Update: {
          adl?: string
          break_even_price?: string
          entry_price?: string
          exchange?: string
          id?: string
          initial_margin?: string
          leverage?: string
          liquidation_price?: string
          maint_margin?: string
          margin_asset?: string
          mark_price?: string
          notional?: string
          open_order_initial_margin?: string
          position_amount?: string
          position_initial_margin?: string
          position_side?: string
          symbol?: string
          unrealized_pnl?: string
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "futures_position_risks_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      margin_interests: {
        Row: {
          asset: string
          exchange: string
          id: string
          interest: string
          interest_rate: string
          isolated_symbol: string
          principle: string
          time: string
          user_id: string
        }
        Insert: {
          asset?: string
          exchange?: string
          id?: string
          interest?: string
          interest_rate?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          user_id: string
        }
        Update: {
          asset?: string
          exchange?: string
          id?: string
          interest?: string
          interest_rate?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "margin_interests_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      margin_liquidations: {
        Row: {
          average_price: string
          exchange: string
          executed_quantity: string
          id: string
          is_isolated: boolean
          order_id: number
          price: string
          quantity: string
          side: string
          symbol: string
          time: string
          time_in_force: string
          user_id: string
        }
        Insert: {
          average_price?: string
          exchange?: string
          executed_quantity?: string
          id?: string
          is_isolated?: boolean
          order_id?: number
          price?: string
          quantity?: string
          side?: string
          symbol?: string
          time?: string
          time_in_force?: string
          user_id: string
        }
        Update: {
          average_price?: string
          exchange?: string
          executed_quantity?: string
          id?: string
          is_isolated?: boolean
          order_id?: number
          price?: string
          quantity?: string
          side?: string
          symbol?: string
          time?: string
          time_in_force?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "margin_liquidations_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      margin_loans: {
        Row: {
          asset: string
          exchange: string
          id: string
          isolated_symbol: string
          principle: string
          time: string
          transaction_id: number
          user_id: string
        }
        Insert: {
          asset?: string
          exchange?: string
          id?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          transaction_id?: number
          user_id: string
        }
        Update: {
          asset?: string
          exchange?: string
          id?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          transaction_id?: number
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "margin_loans_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      margin_repays: {
        Row: {
          asset: string
          exchange: string
          id: string
          isolated_symbol: string
          principle: string
          time: string
          transaction_id: number
          user_id: string
        }
        Insert: {
          asset?: string
          exchange?: string
          id?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          transaction_id?: number
          user_id: string
        }
        Update: {
          asset?: string
          exchange?: string
          id?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          transaction_id?: number
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "margin_repays_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      nav_history_details: {
        Row: {
          available: string
          balance: string
          borrowed: string
          currency: string
          exchange: string
          id: string
          interest: string
          is_isolated: boolean
          is_margin: boolean
          isolated_symbol: string
          locked: string
          net_asset: string
          net_asset_in_btc: string
          net_asset_in_usd: string
          price_in_usd: string
          session: string
          subaccount: string
          time: string
          user_id: string
        }
        Insert: {
          available?: string
          balance?: string
          borrowed?: string
          currency?: string
          exchange?: string
          id?: string
          interest?: string
          is_isolated?: boolean
          is_margin?: boolean
          isolated_symbol?: string
          locked?: string
          net_asset?: string
          net_asset_in_btc?: string
          net_asset_in_usd?: string
          price_in_usd?: string
          session?: string
          subaccount?: string
          time?: string
          user_id: string
        }
        Update: {
          available?: string
          balance?: string
          borrowed?: string
          currency?: string
          exchange?: string
          id?: string
          interest?: string
          is_isolated?: boolean
          is_margin?: boolean
          isolated_symbol?: string
          locked?: string
          net_asset?: string
          net_asset_in_btc?: string
          net_asset_in_usd?: string
          price_in_usd?: string
          session?: string
          subaccount?: string
          time?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "nav_history_details_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      orders: {
        Row: {
          actual_order_id: number
          client_order_id: string
          created_at: string
          exchange: string
          executed_quantity: string | null
          id: string
          is_futures: boolean
          is_isolated: boolean
          is_margin: boolean
          is_working: boolean
          order_id: string
          order_type: string
          order_uuid: string
          price: string
          quantity: string
          side: string
          status: string
          stop_price: string
          strategy_instance_id: string
          symbol: string
          time_in_force: string
          updated_at: string
          user_id: string
        }
        Insert: {
          actual_order_id?: number
          client_order_id?: string
          created_at?: string
          exchange?: string
          executed_quantity?: string | null
          id?: string
          is_futures?: boolean
          is_isolated?: boolean
          is_margin?: boolean
          is_working?: boolean
          order_id: string
          order_type: string
          order_uuid?: string
          price: string
          quantity: string
          side: string
          status: string
          stop_price?: string
          strategy_instance_id?: string
          symbol: string
          time_in_force?: string
          updated_at?: string
          user_id: string
        }
        Update: {
          actual_order_id?: number
          client_order_id?: string
          created_at?: string
          exchange?: string
          executed_quantity?: string | null
          id?: string
          is_futures?: boolean
          is_isolated?: boolean
          is_margin?: boolean
          is_working?: boolean
          order_id?: string
          order_type?: string
          order_uuid?: string
          price?: string
          quantity?: string
          side?: string
          status?: string
          stop_price?: string
          strategy_instance_id?: string
          symbol?: string
          time_in_force?: string
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "sync_orders_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_balances: {
        Row: {
          available: string
          currency: string
          locked: string
          updated_at: string | null
          user_id: string
        }
        Insert: {
          available?: string
          currency: string
          locked?: string
          updated_at?: string | null
          user_id: string
        }
        Update: {
          available?: string
          currency?: string
          locked?: string
          updated_at?: string | null
          user_id?: string
        }
        Relationships: []
      }
      paper_deposits: {
        Row: {
          address: string
          amount: string
          asset: string
          exchange: string
          id: string
          time: string
          txn_id: string
          user_id: string
        }
        Insert: {
          address?: string
          amount?: string
          asset?: string
          exchange?: string
          id?: string
          time?: string
          txn_id?: string
          user_id: string
        }
        Update: {
          address?: string
          amount?: string
          asset?: string
          exchange?: string
          id?: string
          time?: string
          txn_id?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_deposits_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_futures_position_risks: {
        Row: {
          adl: string
          break_even_price: string
          entry_price: string
          exchange: string
          id: string
          initial_margin: string
          leverage: string
          liquidation_price: string
          maint_margin: string
          margin_asset: string
          mark_price: string
          notional: string
          open_order_initial_margin: string
          position_amount: string
          position_initial_margin: string
          position_side: string
          symbol: string
          unrealized_pnl: string
          updated_at: string
          user_id: string
        }
        Insert: {
          adl?: string
          break_even_price?: string
          entry_price?: string
          exchange?: string
          id?: string
          initial_margin?: string
          leverage?: string
          liquidation_price?: string
          maint_margin?: string
          margin_asset?: string
          mark_price?: string
          notional?: string
          open_order_initial_margin?: string
          position_amount?: string
          position_initial_margin?: string
          position_side?: string
          symbol?: string
          unrealized_pnl?: string
          updated_at?: string
          user_id: string
        }
        Update: {
          adl?: string
          break_even_price?: string
          entry_price?: string
          exchange?: string
          id?: string
          initial_margin?: string
          leverage?: string
          liquidation_price?: string
          maint_margin?: string
          margin_asset?: string
          mark_price?: string
          notional?: string
          open_order_initial_margin?: string
          position_amount?: string
          position_initial_margin?: string
          position_side?: string
          symbol?: string
          unrealized_pnl?: string
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_futures_position_risks_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_margin_interests: {
        Row: {
          asset: string
          exchange: string
          id: string
          interest: string
          interest_rate: string
          isolated_symbol: string
          principle: string
          time: string
          user_id: string
        }
        Insert: {
          asset?: string
          exchange?: string
          id?: string
          interest?: string
          interest_rate?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          user_id: string
        }
        Update: {
          asset?: string
          exchange?: string
          id?: string
          interest?: string
          interest_rate?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_margin_interests_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_margin_liquidations: {
        Row: {
          average_price: string
          exchange: string
          executed_quantity: string
          id: string
          is_isolated: boolean
          order_id: number
          price: string
          quantity: string
          side: string
          symbol: string
          time: string
          time_in_force: string
          user_id: string
        }
        Insert: {
          average_price?: string
          exchange?: string
          executed_quantity?: string
          id?: string
          is_isolated?: boolean
          order_id?: number
          price?: string
          quantity?: string
          side?: string
          symbol?: string
          time?: string
          time_in_force?: string
          user_id: string
        }
        Update: {
          average_price?: string
          exchange?: string
          executed_quantity?: string
          id?: string
          is_isolated?: boolean
          order_id?: number
          price?: string
          quantity?: string
          side?: string
          symbol?: string
          time?: string
          time_in_force?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_margin_liquidations_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_margin_loans: {
        Row: {
          asset: string
          exchange: string
          id: string
          isolated_symbol: string
          principle: string
          time: string
          transaction_id: number
          user_id: string
        }
        Insert: {
          asset?: string
          exchange?: string
          id?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          transaction_id?: number
          user_id: string
        }
        Update: {
          asset?: string
          exchange?: string
          id?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          transaction_id?: number
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_margin_loans_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_margin_repays: {
        Row: {
          asset: string
          exchange: string
          id: string
          isolated_symbol: string
          principle: string
          time: string
          transaction_id: number
          user_id: string
        }
        Insert: {
          asset?: string
          exchange?: string
          id?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          transaction_id?: number
          user_id: string
        }
        Update: {
          asset?: string
          exchange?: string
          id?: string
          isolated_symbol?: string
          principle?: string
          time?: string
          transaction_id?: number
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_margin_repays_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_nav_history_details: {
        Row: {
          available: string
          balance: string
          borrowed: string
          currency: string
          exchange: string
          id: string
          interest: string
          is_isolated: boolean
          is_margin: boolean
          isolated_symbol: string
          locked: string
          net_asset: string
          net_asset_in_btc: string
          net_asset_in_usd: string
          price_in_usd: string
          session: string
          subaccount: string
          time: string
          user_id: string
        }
        Insert: {
          available?: string
          balance?: string
          borrowed?: string
          currency?: string
          exchange?: string
          id?: string
          interest?: string
          is_isolated?: boolean
          is_margin?: boolean
          isolated_symbol?: string
          locked?: string
          net_asset?: string
          net_asset_in_btc?: string
          net_asset_in_usd?: string
          price_in_usd?: string
          session?: string
          subaccount?: string
          time?: string
          user_id: string
        }
        Update: {
          available?: string
          balance?: string
          borrowed?: string
          currency?: string
          exchange?: string
          id?: string
          interest?: string
          is_isolated?: boolean
          is_margin?: boolean
          isolated_symbol?: string
          locked?: string
          net_asset?: string
          net_asset_in_btc?: string
          net_asset_in_usd?: string
          price_in_usd?: string
          session?: string
          subaccount?: string
          time?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_nav_history_details_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_orders: {
        Row: {
          actual_order_id: number
          client_order_id: string
          created_at: string
          exchange: string
          executed_quantity: string | null
          id: string
          is_futures: boolean
          is_isolated: boolean
          is_margin: boolean
          is_working: boolean
          order_id: string
          order_type: string
          order_uuid: string
          price: string
          quantity: string
          side: string
          status: string
          stop_price: string
          strategy_instance_id: string
          symbol: string
          time_in_force: string
          updated_at: string
          user_id: string
        }
        Insert: {
          actual_order_id?: number
          client_order_id?: string
          created_at?: string
          exchange?: string
          executed_quantity?: string | null
          id?: string
          is_futures?: boolean
          is_isolated?: boolean
          is_margin?: boolean
          is_working?: boolean
          order_id?: string
          order_type?: string
          order_uuid?: string
          price?: string
          quantity?: string
          side?: string
          status?: string
          stop_price?: string
          strategy_instance_id?: string
          symbol?: string
          time_in_force?: string
          updated_at?: string
          user_id: string
        }
        Update: {
          actual_order_id?: number
          client_order_id?: string
          created_at?: string
          exchange?: string
          executed_quantity?: string | null
          id?: string
          is_futures?: boolean
          is_isolated?: boolean
          is_margin?: boolean
          is_working?: boolean
          order_id?: string
          order_type?: string
          order_uuid?: string
          price?: string
          quantity?: string
          side?: string
          status?: string
          stop_price?: string
          strategy_instance_id?: string
          symbol?: string
          time_in_force?: string
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_orders_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_positions: {
        Row: {
          average_cost: string
          base: string
          base_currency: string
          created_at: string
          exchange: string
          id: string
          net_profit: string | null
          profit: string | null
          quote: string
          quote_currency: string
          side: string
          strategy: string
          strategy_instance_id: string
          symbol: string
          trade_id: number
          traded_at: string
          user_id: string
        }
        Insert: {
          average_cost?: string
          base?: string
          base_currency?: string
          created_at?: string
          exchange?: string
          id?: string
          net_profit?: string | null
          profit?: string | null
          quote?: string
          quote_currency?: string
          side?: string
          strategy?: string
          strategy_instance_id?: string
          symbol?: string
          trade_id: number
          traded_at: string
          user_id: string
        }
        Update: {
          average_cost?: string
          base?: string
          base_currency?: string
          created_at?: string
          exchange?: string
          id?: string
          net_profit?: string | null
          profit?: string | null
          quote?: string
          quote_currency?: string
          side?: string
          strategy?: string
          strategy_instance_id?: string
          symbol?: string
          trade_id?: number
          traded_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_positions_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_profits: {
        Row: {
          average_cost: string
          base_currency: string
          created_at: string
          exchange: string
          fee: string
          fee_currency: string
          fee_in_usd: string | null
          id: string
          is_buyer: boolean
          is_futures: boolean
          is_isolated: boolean
          is_maker: boolean
          is_margin: boolean
          net_profit: string
          net_profit_margin: string
          price: string
          profit: string
          profit_margin: string
          quantity: string
          quote_currency: string
          quote_quantity: string
          side: string
          strategy: string
          strategy_instance_id: string
          symbol: string
          trade_id: number
          traded_at: string
          user_id: string
        }
        Insert: {
          average_cost?: string
          base_currency?: string
          created_at?: string
          exchange?: string
          fee?: string
          fee_currency?: string
          fee_in_usd?: string | null
          id?: string
          is_buyer?: boolean
          is_futures?: boolean
          is_isolated?: boolean
          is_maker?: boolean
          is_margin?: boolean
          net_profit?: string
          net_profit_margin?: string
          price?: string
          profit?: string
          profit_margin?: string
          quantity?: string
          quote_currency?: string
          quote_quantity?: string
          side?: string
          strategy?: string
          strategy_instance_id?: string
          symbol?: string
          trade_id: number
          traded_at: string
          user_id: string
        }
        Update: {
          average_cost?: string
          base_currency?: string
          created_at?: string
          exchange?: string
          fee?: string
          fee_currency?: string
          fee_in_usd?: string | null
          id?: string
          is_buyer?: boolean
          is_futures?: boolean
          is_isolated?: boolean
          is_maker?: boolean
          is_margin?: boolean
          net_profit?: string
          net_profit_margin?: string
          price?: string
          profit?: string
          profit_margin?: string
          quantity?: string
          quote_currency?: string
          quote_quantity?: string
          side?: string
          strategy?: string
          strategy_instance_id?: string
          symbol?: string
          trade_id?: number
          traded_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_profits_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_rewards: {
        Row: {
          created_at: string
          currency: string
          exchange: string
          id: string
          note: string | null
          quantity: string
          reward_type: string
          spent: boolean
          state: string
          user_id: string
          uuid: string
        }
        Insert: {
          created_at?: string
          currency?: string
          exchange?: string
          id?: string
          note?: string | null
          quantity?: string
          reward_type?: string
          spent?: boolean
          state?: string
          user_id: string
          uuid?: string
        }
        Update: {
          created_at?: string
          currency?: string
          exchange?: string
          id?: string
          note?: string | null
          quantity?: string
          reward_type?: string
          spent?: boolean
          state?: string
          user_id?: string
          uuid?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_rewards_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_trades: {
        Row: {
          exchange: string
          fee: string
          fee_currency: string
          id: string
          is_buyer: boolean
          is_futures: boolean
          is_isolated: boolean
          is_maker: boolean
          is_margin: boolean
          order_id: string
          order_uuid: string
          pnl: string | null
          price: string
          quantity: string
          quote_quantity: string | null
          side: string
          strategy: string
          strategy_instance_id: string
          symbol: string
          trade_id: string
          traded_at: string | null
          user_id: string
        }
        Insert: {
          exchange?: string
          fee?: string
          fee_currency?: string
          id?: string
          is_buyer?: boolean
          is_futures?: boolean
          is_isolated?: boolean
          is_maker?: boolean
          is_margin?: boolean
          order_id?: string
          order_uuid?: string
          pnl?: string | null
          price?: string
          quantity?: string
          quote_quantity?: string | null
          side?: string
          strategy?: string
          strategy_instance_id?: string
          symbol?: string
          trade_id?: string
          traded_at?: string | null
          user_id: string
        }
        Update: {
          exchange?: string
          fee?: string
          fee_currency?: string
          id?: string
          is_buyer?: boolean
          is_futures?: boolean
          is_isolated?: boolean
          is_maker?: boolean
          is_margin?: boolean
          order_id?: string
          order_uuid?: string
          pnl?: string | null
          price?: string
          quantity?: string
          quote_quantity?: string | null
          side?: string
          strategy?: string
          strategy_instance_id?: string
          symbol?: string
          trade_id?: string
          traded_at?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_trades_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      paper_withdraws: {
        Row: {
          address: string
          amount: string
          asset: string
          exchange: string
          id: string
          network: string
          time: string
          txn_fee: string
          txn_fee_currency: string
          txn_id: string
          user_id: string
        }
        Insert: {
          address?: string
          amount?: string
          asset?: string
          exchange?: string
          id?: string
          network?: string
          time?: string
          txn_fee?: string
          txn_fee_currency?: string
          txn_id?: string
          user_id: string
        }
        Update: {
          address?: string
          amount?: string
          asset?: string
          exchange?: string
          id?: string
          network?: string
          time?: string
          txn_fee?: string
          txn_fee_currency?: string
          txn_id?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "paper_withdraws_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      positions: {
        Row: {
          average_cost: string
          base: string
          base_currency: string
          created_at: string
          exchange: string
          id: string
          net_profit: string | null
          profit: string | null
          quote: string
          quote_currency: string
          side: string
          strategy: string
          strategy_instance_id: string
          symbol: string
          trade_id: number
          traded_at: string
          user_id: string
        }
        Insert: {
          average_cost?: string
          base?: string
          base_currency?: string
          created_at?: string
          exchange?: string
          id?: string
          net_profit?: string | null
          profit?: string | null
          quote?: string
          quote_currency?: string
          side?: string
          strategy: string
          strategy_instance_id?: string
          symbol: string
          trade_id: number
          traded_at: string
          user_id: string
        }
        Update: {
          average_cost?: string
          base?: string
          base_currency?: string
          created_at?: string
          exchange?: string
          id?: string
          net_profit?: string | null
          profit?: string | null
          quote?: string
          quote_currency?: string
          side?: string
          strategy?: string
          strategy_instance_id?: string
          symbol?: string
          trade_id?: number
          traded_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "positions_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      profits: {
        Row: {
          average_cost: string
          base_currency: string
          created_at: string
          exchange: string
          fee: string
          fee_currency: string
          fee_in_usd: string | null
          id: string
          is_buyer: boolean
          is_futures: boolean
          is_isolated: boolean
          is_maker: boolean
          is_margin: boolean
          net_profit: string
          net_profit_margin: string
          price: string
          profit: string
          profit_margin: string
          quantity: string
          quote_currency: string
          quote_quantity: string
          side: string
          strategy: string
          strategy_instance_id: string
          symbol: string
          trade_id: number
          traded_at: string
          user_id: string
        }
        Insert: {
          average_cost?: string
          base_currency?: string
          created_at?: string
          exchange?: string
          fee?: string
          fee_currency?: string
          fee_in_usd?: string | null
          id?: string
          is_buyer?: boolean
          is_futures?: boolean
          is_isolated?: boolean
          is_maker?: boolean
          is_margin?: boolean
          net_profit?: string
          net_profit_margin?: string
          price?: string
          profit?: string
          profit_margin?: string
          quantity?: string
          quote_currency?: string
          quote_quantity?: string
          side?: string
          strategy: string
          strategy_instance_id?: string
          symbol: string
          trade_id: number
          traded_at: string
          user_id: string
        }
        Update: {
          average_cost?: string
          base_currency?: string
          created_at?: string
          exchange?: string
          fee?: string
          fee_currency?: string
          fee_in_usd?: string | null
          id?: string
          is_buyer?: boolean
          is_futures?: boolean
          is_isolated?: boolean
          is_maker?: boolean
          is_margin?: boolean
          net_profit?: string
          net_profit_margin?: string
          price?: string
          profit?: string
          profit_margin?: string
          quantity?: string
          quote_currency?: string
          quote_quantity?: string
          side?: string
          strategy?: string
          strategy_instance_id?: string
          symbol?: string
          trade_id?: number
          traded_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "profits_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      rewards: {
        Row: {
          created_at: string
          currency: string
          exchange: string
          id: string
          note: string | null
          quantity: string
          reward_type: string
          spent: boolean
          state: string
          user_id: string
          uuid: string
        }
        Insert: {
          created_at?: string
          currency?: string
          exchange?: string
          id?: string
          note?: string | null
          quantity?: string
          reward_type?: string
          spent?: boolean
          state?: string
          user_id: string
          uuid?: string
        }
        Update: {
          created_at?: string
          currency?: string
          exchange?: string
          id?: string
          note?: string | null
          quantity?: string
          reward_type?: string
          spent?: boolean
          state?: string
          user_id?: string
          uuid?: string
        }
        Relationships: [
          {
            foreignKeyName: "rewards_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      strategy_instances: {
        Row: {
          config: Json
          created_at: string
          cross_exchange: boolean
          exchange: string
          futures_config: Json | null
          instance_id: string
          mode: string
          name: string
          sessions: Json | null
          strategy: string
          symbol: string
          updated_at: string
          user_id: string
        }
        Insert: {
          config?: Json
          created_at?: string
          cross_exchange?: boolean
          exchange?: string
          futures_config?: Json | null
          instance_id: string
          mode: string
          name?: string
          sessions?: Json | null
          strategy: string
          symbol?: string
          updated_at?: string
          user_id: string
        }
        Update: {
          config?: Json
          created_at?: string
          cross_exchange?: boolean
          exchange?: string
          futures_config?: Json | null
          instance_id?: string
          mode?: string
          name?: string
          sessions?: Json | null
          strategy?: string
          symbol?: string
          updated_at?: string
          user_id?: string
        }
        Relationships: []
      }
      strategy_registry: {
        Row: {
          category: string
          created_at: string | null
          cross_exchange: boolean | null
          defaults: Json | null
          description: string | null
          display_name: string
          enabled: boolean | null
          exchanges: Json | null
          fields: Json | null
          id: string
          live_only: boolean | null
          requires_futures: boolean | null
          session_roles: Json | null
          sort_order: number | null
          updated_at: string | null
        }
        Insert: {
          category?: string
          created_at?: string | null
          cross_exchange?: boolean | null
          defaults?: Json | null
          description?: string | null
          display_name: string
          enabled?: boolean | null
          exchanges?: Json | null
          fields?: Json | null
          id: string
          live_only?: boolean | null
          requires_futures?: boolean | null
          session_roles?: Json | null
          sort_order?: number | null
          updated_at?: string | null
        }
        Update: {
          category?: string
          created_at?: string | null
          cross_exchange?: boolean | null
          defaults?: Json | null
          description?: string | null
          display_name?: string
          enabled?: boolean | null
          exchanges?: Json | null
          fields?: Json | null
          id?: string
          live_only?: boolean | null
          requires_futures?: boolean | null
          session_roles?: Json | null
          sort_order?: number | null
          updated_at?: string | null
        }
        Relationships: []
      }
      trades: {
        Row: {
          exchange: string
          fee: string
          fee_currency: string
          id: string
          is_buyer: boolean
          is_futures: boolean
          is_isolated: boolean
          is_maker: boolean
          is_margin: boolean
          order_id: string
          order_uuid: string
          pnl: string | null
          price: string
          quantity: string
          quote_quantity: string | null
          side: string
          strategy: string
          strategy_instance_id: string
          symbol: string
          trade_id: string
          traded_at: string | null
          user_id: string
        }
        Insert: {
          exchange?: string
          fee: string
          fee_currency: string
          id?: string
          is_buyer?: boolean
          is_futures?: boolean
          is_isolated?: boolean
          is_maker?: boolean
          is_margin?: boolean
          order_id: string
          order_uuid?: string
          pnl?: string | null
          price: string
          quantity: string
          quote_quantity?: string | null
          side: string
          strategy?: string
          strategy_instance_id?: string
          symbol: string
          trade_id: string
          traded_at?: string | null
          user_id: string
        }
        Update: {
          exchange?: string
          fee?: string
          fee_currency?: string
          id?: string
          is_buyer?: boolean
          is_futures?: boolean
          is_isolated?: boolean
          is_maker?: boolean
          is_margin?: boolean
          order_id?: string
          order_uuid?: string
          pnl?: string | null
          price?: string
          quantity?: string
          quote_quantity?: string | null
          side?: string
          strategy?: string
          strategy_instance_id?: string
          symbol?: string
          trade_id?: string
          traded_at?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "sync_trades_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      user_profiles: {
        Row: {
          avatar_url: string | null
          created_at: string
          display_name: string | null
          email: string
          id: string
          role: string
          updated_at: string
        }
        Insert: {
          avatar_url?: string | null
          created_at?: string
          display_name?: string | null
          email: string
          id: string
          role?: string
          updated_at?: string
        }
        Update: {
          avatar_url?: string | null
          created_at?: string
          display_name?: string | null
          email?: string
          id?: string
          role?: string
          updated_at?: string
        }
        Relationships: []
      }
      withdraws: {
        Row: {
          address: string
          amount: string
          asset: string
          exchange: string
          id: string
          network: string
          time: string
          txn_fee: string
          txn_fee_currency: string
          txn_id: string
          user_id: string
        }
        Insert: {
          address?: string
          amount?: string
          asset?: string
          exchange?: string
          id?: string
          network?: string
          time?: string
          txn_fee?: string
          txn_fee_currency?: string
          txn_id?: string
          user_id: string
        }
        Update: {
          address?: string
          amount?: string
          asset?: string
          exchange?: string
          id?: string
          network?: string
          time?: string
          txn_fee?: string
          txn_fee_currency?: string
          txn_id?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "withdraws_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
    }
    Views: {
      [_ in never]: never
    }
    Functions: {
      [_ in never]: never
    }
    Enums: {
      [_ in never]: never
    }
    CompositeTypes: {
      [_ in never]: never
    }
  }
}

type DatabaseWithoutInternals = Omit<Database, "__InternalSupabase">

type DefaultSchema = DatabaseWithoutInternals[Extract<keyof Database, "public">]

export type Tables<
  DefaultSchemaTableNameOrOptions extends
    | keyof (DefaultSchema["Tables"] & DefaultSchema["Views"])
    | { schema: keyof DatabaseWithoutInternals },
  TableName extends DefaultSchemaTableNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof (DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"] &
        DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Views"])
    : never = never,
> = DefaultSchemaTableNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? (DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"] &
      DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Views"])[TableName] extends {
      Row: infer R
    }
    ? R
    : never
  : DefaultSchemaTableNameOrOptions extends keyof (DefaultSchema["Tables"] &
        DefaultSchema["Views"])
    ? (DefaultSchema["Tables"] &
        DefaultSchema["Views"])[DefaultSchemaTableNameOrOptions] extends {
        Row: infer R
      }
      ? R
      : never
    : never

export type TablesInsert<
  DefaultSchemaTableNameOrOptions extends
    | keyof DefaultSchema["Tables"]
    | { schema: keyof DatabaseWithoutInternals },
  TableName extends DefaultSchemaTableNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"]
    : never = never,
> = DefaultSchemaTableNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"][TableName] extends {
      Insert: infer I
    }
    ? I
    : never
  : DefaultSchemaTableNameOrOptions extends keyof DefaultSchema["Tables"]
    ? DefaultSchema["Tables"][DefaultSchemaTableNameOrOptions] extends {
        Insert: infer I
      }
      ? I
      : never
    : never

export type TablesUpdate<
  DefaultSchemaTableNameOrOptions extends
    | keyof DefaultSchema["Tables"]
    | { schema: keyof DatabaseWithoutInternals },
  TableName extends DefaultSchemaTableNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"]
    : never = never,
> = DefaultSchemaTableNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? DatabaseWithoutInternals[DefaultSchemaTableNameOrOptions["schema"]]["Tables"][TableName] extends {
      Update: infer U
    }
    ? U
    : never
  : DefaultSchemaTableNameOrOptions extends keyof DefaultSchema["Tables"]
    ? DefaultSchema["Tables"][DefaultSchemaTableNameOrOptions] extends {
        Update: infer U
      }
      ? U
      : never
    : never

export type Enums<
  DefaultSchemaEnumNameOrOptions extends
    | keyof DefaultSchema["Enums"]
    | { schema: keyof DatabaseWithoutInternals },
  EnumName extends DefaultSchemaEnumNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof DatabaseWithoutInternals[DefaultSchemaEnumNameOrOptions["schema"]]["Enums"]
    : never = never,
> = DefaultSchemaEnumNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? DatabaseWithoutInternals[DefaultSchemaEnumNameOrOptions["schema"]]["Enums"][EnumName]
  : DefaultSchemaEnumNameOrOptions extends keyof DefaultSchema["Enums"]
    ? DefaultSchema["Enums"][DefaultSchemaEnumNameOrOptions]
    : never

export type CompositeTypes<
  PublicCompositeTypeNameOrOptions extends
    | keyof DefaultSchema["CompositeTypes"]
    | { schema: keyof DatabaseWithoutInternals },
  CompositeTypeName extends PublicCompositeTypeNameOrOptions extends {
    schema: keyof DatabaseWithoutInternals
  }
    ? keyof DatabaseWithoutInternals[PublicCompositeTypeNameOrOptions["schema"]]["CompositeTypes"]
    : never = never,
> = PublicCompositeTypeNameOrOptions extends {
  schema: keyof DatabaseWithoutInternals
}
  ? DatabaseWithoutInternals[PublicCompositeTypeNameOrOptions["schema"]]["CompositeTypes"][CompositeTypeName]
  : PublicCompositeTypeNameOrOptions extends keyof DefaultSchema["CompositeTypes"]
    ? DefaultSchema["CompositeTypes"][PublicCompositeTypeNameOrOptions]
    : never

export const Constants = {
  public: {
    Enums: {},
  },
} as const
