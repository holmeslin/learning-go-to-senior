# Summary

[前言](foreword.md)

# 第一部　程式碼設計

- [套件與 API 設計](chapter1/README.md)
    - [套件切分的依據（依職責，不依型別）](chapter1/01_package_by_responsibility.md)
    - [套件命名與 `util` 陷阱](chapter1/02_package_naming.md)
    - [什麼時候用 `internal`，什麼時候該公開]()
    - [匯出名稱與 API 穩定性](chapter1/04_exported_api_stability.md)
    - [建構函式與 functional options]()
    - [什麼時候讓零值可用，什麼時候該強制用建構函式]()
    - [循環相依與依賴方向](chapter1/07_import_cycle.md)
    - [棄用（deprecation）與相容性]()
- [錯誤設計]()
- [抽象的取捨]()
- [資料存取的設計]()

# 第二部　測試

- [測試策略]()

# 第三部　並行實務

- [結構化並行]()

# 第四部　效能

- [效能調查流程]()
- [runtime]()

# 第五部　上線與維運

- [可觀測性]()
- [韌性]()
- [安全實務]()
- [建置與發布]()

# 第六部　在團隊裡寫 Go

- [code review]()
- [設計文件]()
- [事故處理]()

# 第七部　綜合實戰

- [一個服務從設計到上線]()
