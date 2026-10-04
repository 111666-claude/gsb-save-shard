// Command save-shard 跑存档分片样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/shard"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("save-shard", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "size", "size / dup / missing / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	switch *sample {
	case "size":
		store := shard.New(4)
		fmt.Fprintf(stdout, "ok=%v\n", store.Put("slot1", 0, "ab", "r1"))
	case "dup":
		store := shard.New(4)
		store.Put("slot1", 0, "abcd", "same")
		fmt.Fprintf(stdout, "second=%v\n", store.Put("slot1", 1, "efgh", "same"))
	case "missing":
		store := shard.New(4)
		store.Put("slot1", 0, "abcd", "r1")
		err := store.Commit("slot1")
		fmt.Fprintf(stdout, "err=%v committed=%v\n", err != nil, store.Committed("slot1"))
	case "scan":
		store := shard.New(4)
		for index := 0; index < 3000; index++ {
			store.Put(fmt.Sprintf("slot-%d", index), 0, "abcd", fmt.Sprintf("r-%d", index))
		}
		for index := 0; index < 3000; index++ {
			store.Commit("slot-2999")
		}
		fmt.Fprintf(stdout, "scanned=%d\n", store.Scanned())
	default:
		fmt.Fprintln(stderr, "需要 --sample size|dup|missing|scan")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
