-- Create storage bucket for backtest reports
INSERT INTO storage.buckets (id, name, public)
VALUES ('backtest-reports', 'backtest-reports', false)
ON CONFLICT (id) DO NOTHING;

-- Allow users to read their own backtest reports via signed URLs
CREATE POLICY "Users can read own backtest reports"
  ON storage.objects
  FOR SELECT
  TO authenticated
  USING (
    bucket_id = 'backtest-reports'
    AND (storage.foldername(name))[1] = auth.uid()::text
  );

-- Allow service role full access
CREATE POLICY "Service role full access to backtest reports"
  ON storage.objects
  FOR ALL
  TO service_role
  USING (bucket_id = 'backtest-reports')
  WITH CHECK (bucket_id = 'backtest-reports');
