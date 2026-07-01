package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/you/glass"
)

func main() {
	fmt.Println("demo pid:", os.Getpid())
	fmt.Println("run: glass attach", os.Getpid())

	go func() {
		for {
			time.Sleep(2 * time.Second)
			go func() {
				select {}
			}()
		}
	}()

	go func() {
		for {
			time.Sleep(3 * time.Second)
			if rand.Intn(3) == 0 {
				glass.RecordError(errors.New("payment gateway timeout: connection reset by peer"))
			}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	http.ListenAndServe(":8080", nil)
}
