package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jessevdk/go-flags"
)

var cmdArgs struct {
	// Slice of bool will append 'true' each time the option
	// is encountered (can be set multiple times, like -vvv)
	Ignore *string `long:"ignore" description:"Commar-seperated list of directories to ignore; (defaults: lib,inc"`

	Input string `short:"i" long:"input" required:"true" description:"Input directory for SASS/SCSS files" default:"sass"`

	Output string `short:"o" long:"output" description:"Output directory for CSS files" default:"css"`

	Loop bool `short:"l" long:"watch" description:"Loop"`

	LoopInterval int64 `long:"interval" description:"Loop interval in miliseconds" default:"2000"`
}

var (
	ignoreDirs = []string{"lib", "inc"}
	wd         string
	walkWg     sync.WaitGroup
	sassDir    string
	cssDir     string
)

func init() {
	_, err := flags.Parse(&cmdArgs)
	if err != nil {
		flagsError, ok := err.(*flags.Error)
		if ok && flagsError.Type == flags.ErrHelp {
			os.Exit(0)
		}
		log.Fatal(err)
	}

	if cmdArgs.Ignore != nil {
		ignoreDirs = strings.Split(*cmdArgs.Ignore, ",")
	}
}

func main() {
	var err error
	wd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	sassDir = filepath.Join(wd, cmdArgs.Input)
	cssDir = filepath.Join(wd, cmdArgs.Output)

	for {
		convertSASSToCSS()

		fmt.Println("================")
		fmt.Println("      Done      ")
		fmt.Println("================")

		if !cmdArgs.Loop {
			break
		}

		time.Sleep(time.Millisecond * time.Duration(cmdArgs.LoopInterval))
	}
}

func convertSASSToCSS() {
	filepath.Walk(sassDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			for _, dir := range ignoreDirs {
				if info.Name() == dir {
					return filepath.SkipDir
				}
			}
		}

		if !info.IsDir() {
			walkWg.Add(1)
			go func(path string) {
				defer walkWg.Done()
				localPath := filepath.ToSlash(strings.TrimPrefix(path, sassDir+string(os.PathSeparator)))

				localPathDirs := strings.Split(filepath.ToSlash(filepath.Join(cssDir, localPath)), "/")
				if len(localPathDirs) > 1 {
					localPathDirsStr := strings.Join(localPathDirs[:len(localPathDirs)-1], "/")
					os.MkdirAll(localPathDirsStr, os.ModePerm)
				}

				inPath := filepath.ToSlash(path)
				outPath := strings.TrimSuffix(filepath.ToSlash(filepath.Join(cssDir, localPath)), ".sass") + ".css"

				fmt.Println(inPath, "->", outPath)

				exec.Command(
					"sass", "--compass", "--style", "compressed", "--sourcemap=none", "--no-cache",
					inPath,
					outPath,
				).Run()
			}(path)
		}
		return nil
	})
	walkWg.Wait()
}
