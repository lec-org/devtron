-- 不截断已有凭据；仅在全部密文符合旧字段限制时允许回退。
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM public.git_provider WHERE length(access_token) > 250) THEN
        RAISE EXCEPTION '现有 Git 凭据超过旧字段上限，请先移除组织 App 账户再回退';
    END IF;
END $$;
ALTER TABLE public.git_provider ALTER COLUMN access_token TYPE varchar(250);
