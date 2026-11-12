-- GitHub App 的临时凭据经过加密后可能超过旧字段的 250 字符上限。
ALTER TABLE public.git_provider ALTER COLUMN access_token TYPE text;
