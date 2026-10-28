-- 先擴充：新欄位有預設值，還沒更新的舊版程式 INSERT 不帶 name 也照常成功。
ALTER TABLE users ADD COLUMN name TEXT NOT NULL DEFAULT '';
