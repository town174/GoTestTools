package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var verbose = flag.Bool("v", false, "show verbose progress messages")
var sendMsg = make(chan struct{}, 50)
var done = make(chan struct{})

//获取目录下文件信息
func dirents(dir string) []os.FileInfo {
	select {
	case sendMsg <- struct{}{}:
	case <-done:
		return nil
	}
	//why
	defer func() { <-sendMsg }()
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dul:%v\n", err)
		return nil
	}
	return entries
}

//递归调用dirents函数
func walkDir(dir string, n *sync.WaitGroup, fileSizes chan<- int64) {
	defer n.Done()
	if cancelled() {
		return
	}
	for _, entry := range dirents(dir) {
		if entry.IsDir() {
			n.Add(1)
			subdir := filepath.Join(dir, entry.Name())
			go walkDir(subdir, n, fileSizes)
		} else {
			fileSizes <- entry.Size()
		}
	}
}

func printDiskUsage(nfiles, nbytes int64) {
	fmt.Printf("%d files %.1f GB\n", nfiles, float64(nbytes/1e9))
}

func cancelled() bool {
	select {
	case <-done:
		return true
	default:
		return false

	}
}

func main() {
	flag.Parse()
	roots := flag.Args()
	var tick <-chan time.Time

	if *verbose {
		tick = time.Tick(time.Microsecond * 500)
	}
	if len(roots) == 0 {
		roots = []string{"."}
	}

	fileSizes := make(chan int64)
	var nfiles, nbytes int64
	//why
	var n sync.WaitGroup

	for _, root := range roots {
		n.Add(1)
		go walkDir(root, &n, fileSizes)
	}
	go func() {
		n.Wait()
		close(fileSizes)
	}()
	go func() {
		os.Stdin.Read(make([]byte, 1))
		close(done)
	}()
}
