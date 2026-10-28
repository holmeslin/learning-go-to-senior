# 抽象的取捨

前作已經教過〈[介面](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/01_interface.html)〉、〈[嵌入](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter3/12_embedding.html)〉與〈[泛型函式](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter6/08_generic_function.html)〉的語法，也談過〈[小介面設計](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter4/10_small_interfaces.html)〉和附錄二〈[介面 vs 泛型的取捨](https://holmeslin.github.io/learning-go-from-zero/zh-TW/appendix2/e_interface_vs_generics.html)〉的基本原則。會用這些工具的人，在真實專案裡常走向兩個極端：一種是什麼都先抽象，每個 service 配一個只有一個實作的介面，建構函式回傳介面，兩段長得像的程式碼立刻合成泛型，struct 隨手嵌入別人的型別；另一種是完全不抽象，到處直接依賴具體實作，測試只能連真的資料庫。

這一章的主軸是：抽象有成本。每多一層間接，讀程式的人就多跳一次；每個匯出的介面或嵌入，都是日後要維持的承諾。所以這一章不是在教更多語法，每一集都會讓你同時看到抽象的代價與收益，再依情境決定要不要加、加在哪裡、加多大。
