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
      bots: {
        Row: {
          bbgo_pid: number | null
          config: Json
          config_path: string | null
          created_at: string
          exchange: string
          grpc_port: number | null
          id: string
          mode: string
          name: string
          status: string
          strategy: string
          updated_at: string
          user_id: string
          webserver_port: number | null
        }
        Insert: {
          bbgo_pid?: number | null
          config?: Json
          config_path?: string | null
          created_at?: string
          exchange: string
          grpc_port?: number | null
          id?: string
          mode?: string
          name: string
          status?: string
          strategy: string
          updated_at?: string
          user_id: string
          webserver_port?: number | null
        }
        Update: {
          bbgo_pid?: number | null
          config?: Json
          config_path?: string | null
          created_at?: string
          exchange?: string
          grpc_port?: number | null
          id?: string
          mode?: string
          name?: string
          status?: string
          strategy?: string
          updated_at?: string
          user_id?: string
          webserver_port?: number | null
        }
        Relationships: [
          {
            foreignKeyName: "bots_user_id_fkey"
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
      sync_cursors: {
        Row: {
          bot_id: string
          id: string
          last_synced_at: string
          last_synced_id: string
          table_name: string
        }
        Insert: {
          bot_id: string
          id?: string
          last_synced_at?: string
          last_synced_id: string
          table_name: string
        }
        Update: {
          bot_id?: string
          id?: string
          last_synced_at?: string
          last_synced_id?: string
          table_name?: string
        }
        Relationships: [
          {
            foreignKeyName: "sync_cursors_bot_id_fkey"
            columns: ["bot_id"]
            isOneToOne: false
            referencedRelation: "bots"
            referencedColumns: ["id"]
          },
        ]
      }
      sync_orders: {
        Row: {
          bot_id: string | null
          created_at: string
          creation_time: string | null
          executed_quantity: string | null
          id: string
          order_id: string
          price: string
          quantity: string
          side: string
          status: string
          symbol: string
          synced_at: string
          type: string
          user_id: string
        }
        Insert: {
          bot_id?: string | null
          created_at?: string
          creation_time?: string | null
          executed_quantity?: string | null
          id?: string
          order_id: string
          price: string
          quantity: string
          side: string
          status: string
          symbol: string
          synced_at?: string
          type: string
          user_id: string
        }
        Update: {
          bot_id?: string | null
          created_at?: string
          creation_time?: string | null
          executed_quantity?: string | null
          id?: string
          order_id?: string
          price?: string
          quantity?: string
          side?: string
          status?: string
          symbol?: string
          synced_at?: string
          type?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "sync_orders_bot_id_fkey"
            columns: ["bot_id"]
            isOneToOne: false
            referencedRelation: "bots"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "sync_orders_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      sync_trades: {
        Row: {
          bot_id: string | null
          created_at: string
          fee: string
          fee_currency: string
          id: string
          order_id: string
          pnl: string | null
          price: string
          quantity: string
          quote_quantity: string | null
          side: string
          symbol: string
          synced_at: string
          trade_id: string
          traded_at: string | null
          user_id: string
        }
        Insert: {
          bot_id?: string | null
          created_at?: string
          fee: string
          fee_currency: string
          id?: string
          order_id: string
          pnl?: string | null
          price: string
          quantity: string
          quote_quantity?: string | null
          side: string
          symbol: string
          synced_at?: string
          trade_id: string
          traded_at?: string | null
          user_id: string
        }
        Update: {
          bot_id?: string | null
          created_at?: string
          fee?: string
          fee_currency?: string
          id?: string
          order_id?: string
          pnl?: string | null
          price?: string
          quantity?: string
          quote_quantity?: string | null
          side?: string
          symbol?: string
          synced_at?: string
          trade_id?: string
          traded_at?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "sync_trades_bot_id_fkey"
            columns: ["bot_id"]
            isOneToOne: false
            referencedRelation: "bots"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "sync_trades_user_id_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "user_profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      user_containers: {
        Row: {
          created_at: string
          mode: string
          status: string
          strategies: Json
          updated_at: string
          user_id: string
        }
        Insert: {
          created_at?: string
          mode?: string
          status?: string
          strategies?: Json
          updated_at?: string
          user_id: string
        }
        Update: {
          created_at?: string
          mode?: string
          status?: string
          strategies?: Json
          updated_at?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "user_containers_user_id_fkey"
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
