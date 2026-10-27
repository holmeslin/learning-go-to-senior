// 書中範例共用的 go.mod：checkcode 會把它和 go.sum 複製到每個範例的暫存目錄。
// 範例需要第三方套件時，在這裡鎖定版本；目前沒有任何相依。
module example

go 1.27
