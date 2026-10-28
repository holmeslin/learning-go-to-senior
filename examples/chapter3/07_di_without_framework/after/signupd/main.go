// Command signupd 註冊命令列給的信箱，示範在 main 裡把整張物件圖手動接起來。
package main

import (
	"log"
	"os"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter3/07_di_without_framework/after/signup"
)

// ANCHOR: main

func main() {
	// 接線只發生在這裡：從最底層的相依開始，一層一層交給上一層。
	store := signup.NewStore()
	mailer := signup.NewSMTPMailer("smtp.example.com:25")
	svc := signup.NewService(store, mailer)

	for _, email := range os.Args[1:] {
		if err := svc.Register(email); err != nil {
			log.Printf("register %s: %v", email, err)
		}
	}
}

// ANCHOR_END: main
