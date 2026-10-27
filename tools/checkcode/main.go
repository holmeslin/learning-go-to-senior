// checkcode 抽出書中所有 ```go 程式碼區塊並驗證。
//
// 用法：go run ./tools/checkcode zh-TW/src
//
// info string 標註見 README.md。
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// depsDir 放書中範例共用的 go.mod 與 go.sum；checkcode 必須在 repo 根目錄執行。
const depsDir = "tools/checkcode/deps"

type block struct {
	file  string
	line  int
	code  string
	attrs map[string]string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: checkcode <dir>")
		os.Exit(2)
	}
	blocks, err := collect(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// 先編譯一次共用相依，讓建置快取就緒；
	// 否則冷快取時多個範例會同時編譯它而超過單一範例的逾時。
	warm := exec.Command("go", "build", "./...")
	warm.Dir = depsDir
	if out, err := warm.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build deps: %v\n%s", err, out)
		os.Exit(2)
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		failures []string
		checked  int
	)
	sem := make(chan struct{}, runtime.NumCPU())
	for _, b := range blocks {
		if _, ok := b.attrs["ignore"]; ok {
			continue
		}
		checked++
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := check(b); err != nil {
				mu.Lock()
				failures = append(failures, fmt.Sprintf("%s:%d: %v", b.file, b.line, err))
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	for _, f := range failures {
		fmt.Println("FAIL", f)
	}
	fmt.Printf("%d blocks checked, %d failed, %d ignored\n", checked, len(failures), len(blocks)-checked)
	if len(failures) > 0 {
		os.Exit(1)
	}
}

func collect(root string) ([]block, error) {
	var blocks []block
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		var cur *block
		var buf strings.Builder
		n := 0
		for sc.Scan() {
			n++
			line := sc.Text()
			trimmed := strings.TrimSpace(line)
			switch {
			case cur == nil && strings.HasPrefix(trimmed, "```go"):
				info := strings.TrimPrefix(trimmed, "```")
				if info != "go" && !strings.HasPrefix(info, "go,") {
					continue // 例如 ```gomod
				}
				cur = &block{file: path, line: n, attrs: parseAttrs(info)}
				buf.Reset()
			case cur != nil && trimmed == "```":
				cur.code = buf.String()
				blocks = append(blocks, *cur)
				cur = nil
			case cur != nil:
				buf.WriteString(line)
				buf.WriteByte('\n')
			}
		}
		if cur != nil {
			return fmt.Errorf("%s:%d: unterminated code block", path, cur.line)
		}
		return sc.Err()
	})
	return blocks, err
}

func parseAttrs(info string) map[string]string {
	attrs := map[string]string{}
	for _, a := range strings.Split(info, ",")[1:] {
		k, v, _ := strings.Cut(a, "=")
		attrs[k] = strings.ReplaceAll(v, `\n`, "\n")
	}
	return attrs
}

func check(b block) error {
	dir, err := os.MkdirTemp("", "checkcode")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	files := map[string][]byte{"main.go": []byte(b.code)}
	for _, name := range []string{"go.mod", "go.sum"} {
		if files[name], err = os.ReadFile(filepath.Join(depsDir, name)); err != nil {
			return err
		}
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
			return err
		}
	}

	if _, ok := b.attrs["compile_fail"]; ok {
		if _, err := goCmd(dir, "", "build", "-o", os.DevNull, "."); err == nil {
			return errors.New("expected compile error, but it compiled")
		}
		return nil
	}
	if out, err := goCmd(dir, "", "vet", "."); err != nil {
		return fmt.Errorf("go vet: %v\n%s", err, out)
	}
	if _, ok := b.attrs["norun"]; ok {
		return nil
	}

	bin := filepath.Join(dir, "prog")
	if out, err := goCmd(dir, "", "build", "-o", bin, "."); err != nil {
		return fmt.Errorf("go build: %v\n%s", err, out)
	}
	wantExit := 0
	if v, ok := b.attrs["exit"]; ok {
		if wantExit, err = strconv.Atoi(v); err != nil {
			return fmt.Errorf("bad exit attr %q", v)
		}
	}
	out, err := run(dir, b.attrs["stdin"], bin)
	gotExit := 0
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		gotExit = ee.ExitCode()
	} else if err != nil {
		return fmt.Errorf("run: %v\n%s", err, out)
	}
	if gotExit != wantExit {
		return fmt.Errorf("exit code %d, want %d\n%s", gotExit, wantExit, out)
	}
	return nil
}

func goCmd(dir, stdin string, args ...string) ([]byte, error) {
	return run(dir, stdin, "go", args...)
}

// run 執行指令，逾時 30 秒。
func run(dir, stdin, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if ctx.Err() != nil {
		return out.Bytes(), errors.New("timeout")
	}
	return out.Bytes(), err
}
