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
      strategy_instances: {
        Row: {
          config: Json
          created_at: string
          cross_exchange: boolean
          exchange: string
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
