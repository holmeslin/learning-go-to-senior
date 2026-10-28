# Summary

[前言](foreword.md)

# 第一部　程式碼設計

- [套件與 API 設計](chapter1/README.md)
    - [套件切分的依據（依職責，不依型別）](chapter1/01_package_by_responsibility.md)
    - [套件命名與 `util` 陷阱](chapter1/02_package_naming.md)
    - [什麼時候用 `internal`，什麼時候該公開](chapter1/03_internal_or_public.md)
    - [匯出名稱與 API 穩定性](chapter1/04_exported_api_stability.md)
    - [建構函式與 functional options](chapter1/05_constructor_functional_options.md)
    - [什麼時候讓零值可用，什麼時候該強制用建構函式](chapter1/06_zero_value_or_constructor.md)
    - [循環相依與依賴方向](chapter1/07_import_cycle.md)
    - [棄用（deprecation）與相容性](chapter1/08_deprecation_and_compatibility.md)
- [錯誤設計](chapter2/README.md)
    - [哨兵錯誤、自訂型別、不透明錯誤的選擇](chapter2/01_sentinel_custom_opaque.md)
    - [包裝錯誤：`%w` 該不該暴露給呼叫端](chapter2/02_wrap_or_expose.md)
    - [在哪一層處理、哪一層只回傳](chapter2/03_handle_at_boundary.md)
    - [錯誤訊息怎麼寫](chapter2/04_error_message_style.md)
    - [多個錯誤要合併回傳，還是只回第一個](chapter2/05_join_or_first_error.md)
    - [什麼時候可以 `panic`，什麼時候一定要回傳 `error`](chapter2/06_panic_or_error.md)
    - [錯誤是 API 契約的一部分](chapter2/07_error_contract.md)
- [抽象的取捨]()
    - [接受介面、回傳 struct](chapter3/02_accept_interfaces_return_structs.md)
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
