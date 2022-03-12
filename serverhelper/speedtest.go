package serverhelper

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func SpeedTestHelper(w http.ResponseWriter, r *http.Request) {
	sizeStr := r.URL.Query().Get("size")
	size := 10
	switch sizeStr {
	case "10":
		size = 10
	case "20":
		size = 20
	case "30":
		size = 30
	case "50":
		size = 50
	case "100":
		size = 100
	}

	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size*1024*1024))

	buf := make([]byte, 1024*16)
	for i := 0; i < 1024*16; i++ {
		buf[i] = byte(rand.Int() % 256)
	}

	for i := 0; i < size*64; i++ {
		w.Write(buf)
	}
}
