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
- [抽象的取捨](chapter3/README.md)
    - [在使用端定義介面](chapter3/01_define_interface_at_consumer.md)
    - [接受介面、回傳 struct](chapter3/02_accept_interfaces_return_structs.md)
    - [介面何時是多餘的](chapter3/03_redundant_interface.md)
    - [什麼時候用泛型、什麼時候用介面、什麼時候複製貼上](chapter3/04_generics_interface_or_copy.md)
    - [過早抽象的代價](chapter3/05_premature_abstraction.md)
    - [什麼時候用嵌入，什麼時候該明確委派](chapter3/06_embedding_or_delegation.md)
    - [不用框架的相依注入](chapter3/07_di_without_framework.md)
- [資料存取的設計](chapter4/README.md)
    - [要不要有 repository 這一層](chapter4/01_repository_layer.md)
    - [交易邊界放在哪裡](chapter4/02_transaction_boundary.md)
    - [查詢逾時該設在哪一層](chapter4/03_query_timeout_layer.md)
    - [連線池要設多大、設錯會怎樣](chapter4/04_pool_size.md)
    - [N+1 與查詢組裝](chapter4/05_n_plus_one.md)
    - [migration](chapter4/06_migration.md)
    - [測試資料層：fake 與真實資料庫](chapter4/07_fake_or_real_database.md)

# 第二部　測試

- [測試策略](chapter5/README.md)
    - [測什麼、不測什麼](chapter5/01_what_to_test.md)
    - [表格驅動測試的邊界](chapter5/02_table_driven_boundary.md)
    - [fake、stub、mock 的選擇](chapter5/03_fake_stub_mock.md)
    - [測試替身放在哪裡](chapter5/04_where_test_doubles_live.md)
    - [整合測試與 build tags](chapter5/05_integration_tests_build_tags.md)
    - [golden file](chapter5/06_golden_file.md)
    - [要啟動真的 server，還是只測 handler](chapter5/09_server_or_handler.md)
    - [不穩定（flaky）測試](chapter5/10_flaky_tests.md)

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
