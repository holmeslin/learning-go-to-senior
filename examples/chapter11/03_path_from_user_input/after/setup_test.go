package reports_test

import (
	"os"
	"path/filepath"
)

// secret 是放在報表目錄之外的檔案內容；它出現在輸出裡，就代表檔案被讀到了。
const secret = "example-not-a-real-secret"

// attack 是一個要拿去試的檔名。絕對路徑含暫存目錄，要等目錄建好才知道。
type attack struct {
	label string
	name  string
}

// setup 在 base 底下建出報表目錄 reports，和它旁邊名字開頭相同的 reports-private，
// 回傳報表目錄與要試的檔名。
func setup(base string) (dir string, attacks []attack, err error) {
	dir = filepath.Join(base, "reports")
	private := filepath.Join(base, "reports-private")
	files := map[string]string{
		filepath.Join(dir, "sales.csv"):         "region,total\n",
		filepath.Join(dir, "archive", "q1.csv"): "region,total\n",
		filepath.Join(dir, ".state"):            "cursor=42",
		filepath.Join(private, "payroll.csv"):   secret,
		filepath.Join(base, "outside.csv"):      secret,
	}
	for path, content := range files {
		if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", nil, err
		}
		if err = os.WriteFile(path, []byte(content), 0o644); err != nil {
			return "", nil, err
		}
	}
	// 報表目錄裡的一個符號連結，指到目錄外面。
	link := filepath.Join("..", "reports-private", "payroll.csv")
	if err = os.Symlink(link, filepath.Join(dir, "link.csv")); err != nil {
		return "", nil, err
	}
	return dir, []attack{
		{"正常檔名", "sales.csv"},
		{"子目錄", "archive/q1.csv"},
		{"..", "../outside.csv"},
		{".. 加相同開頭的目錄", "../reports-private/payroll.csv"},
		{"絕對路徑", filepath.Join(private, "payroll.csv")},
		{"符號連結", "link.csv"},
	}, nil
}
